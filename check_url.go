package main

import (
	"fmt"
	"log"
	"net/http"
	nurl "net/url"
	"os"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/sync/errgroup"
)

var (
	errTimeOut = fmt.Errorf("head request time out")
)

const (
	SCHEME_MAILTO = "mailto"
	SCHEME_TEL    = "tel"
	SCHEME_CALLTO = "callto"
)

func headRequest(client *http.Client, url string) (*http.Response, error) {
	resp, err := client.Head(url)
	if err != nil {
		if os.IsTimeout(err) {
			return nil, errTimeOut
		} else {
			fmt.Println(err)
			return nil, err
		}
	}
	return resp, nil
}

func getRequest(client *http.Client, url string) (*http.Response, error) {
	resp, err := client.Get(url)
	if err != nil {
		if os.IsTimeout(err) {
			return nil, errTimeOut
		} else {
			fmt.Println(err)
			return nil, err
		}
	}
	return resp, nil
}

func checkUrl(base nurl.URL, url string, urlTree *UrlTreeStruct, progress func(string, float64), index, count float64) {
	uts := urlTree.FindByUrl(url)
	if uts == nil {
		fmt.Println("Uts not found!")
		return
	}
	client := &http.Client{
		Timeout: time.Second * 10,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}
	resp, err := getRequest(client, url)
	if err != nil {
		switch err {
		case errTimeOut:
			uts.Status = STATUS_LONGWAIT
			fmt.Println("To long to wait")
			return
		default:
			uts.Status = STATUS_FAILURE
			return
		}
	}
	statCode := resp.StatusCode
	fmt.Println("Code of", url, "is", statCode)
	if statCode == 200 {
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			uts.Status = STATUS_PROBLEM
			fmt.Println("Can not read body")
			return
		}
		fmt.Println("Size of", url, "document is", doc.Length())
		as := doc.Find("a")
		asCount := float64(as.Length())
		partCoeff := 1 / asCount / count
		group := new(errgroup.Group)
		group.SetLimit(max_inner_pool)
		as.Each(func(i int, a *goquery.Selection) {
			if href, ok := a.Attr("href"); ok {
				if err == nil {
					group.Go(func() error {
						checkInnerUrl(base, href, uts)
						progress(fmt.Sprintf("Checked inner %s", href), (partCoeff*float64(i)+index)/count)
						return nil
					})
				} else {
					fmt.Println("have no href? ", err)
				}
			}
		})
		group.Wait()
		uts.Status = STATUS_SUCCESS
		return
	} else if statCode >= 301 && statCode <= 308 {
		newUrl, err := resp.Location()
		if err != nil {
			checkUrl(base, newUrl.String(), urlTree, progress, index, count)
			return
		}
	}
	uts.Status = STATUS_FAILURE
}

func configureAndBindInnerUrl(url string, status, linkType int, sourceSize int64, urlContainer *UrlTreeStruct) {
	urlElement := NewUrlStruct(url)
	urlElement.Status = status
	urlElement.LinkType = linkType
	urlElement.SourceSize = sourceSize
	urlContainer.AppendInnerUrl(urlElement)
}

func checkInnerUrl(base nurl.URL, url string, urlContainer *UrlTreeStruct) {
	based_url, err := base.Parse(url)
	if err != nil {
		fmt.Println("try base failed:", url)
		configureAndBindInnerUrl(url, STATUS_FAILURE, LINK_TYPE_PAGE, -1, urlContainer)
		return
	}
	str_based_url := based_url.String()
	//fmt.Println("Sceme of", url, "is", part_url.Scheme)
	switch based_url.Scheme {
	case SCHEME_MAILTO:
		configureAndBindInnerUrl(str_based_url, STATUS_PROBLEM, LINK_TYPE_MAILTO, -1, urlContainer)
		return
	case SCHEME_TEL:
		configureAndBindInnerUrl(str_based_url, STATUS_PROBLEM, LINK_TYPE_TEL, -1, urlContainer)
		return
	case SCHEME_CALLTO:
		configureAndBindInnerUrl(str_based_url, STATUS_PROBLEM, LINK_TYPE_CALLTO, -1, urlContainer)
		return
	}
	client := &http.Client{
		Timeout: time.Second * 20,
	}
	resp, err := headRequest(client, str_based_url)
	if err != nil {
		switch err {
		case errTimeOut:
			fmt.Println("To long to wait")
			configureAndBindInnerUrl(str_based_url, STATUS_LONGWAIT, LINK_TYPE_PAGE, -1, urlContainer)
			return
		default:
			configureAndBindInnerUrl(str_based_url, STATUS_FAILURE, LINK_TYPE_PAGE, -1, urlContainer)
			return
		}
	}
	statCode := resp.StatusCode
	contentLen := resp.ContentLength
	fmt.Println("Code of inner", url, "is", statCode)
	if statCode == 200 {
		configureAndBindInnerUrl(str_based_url, STATUS_SUCCESS, LINK_TYPE_PAGE, contentLen, urlContainer)
		return
	} else if statCode >= 300 && statCode <= 308 {
		newUrl, err := resp.Location()
		if err != nil {
			checkInnerUrl(base, newUrl.String(), urlTree)
			return
		}
	} else if statCode == 404 {
		configureAndBindInnerUrl(str_based_url, STATUS_NOTFOUND, LINK_TYPE_PAGE, contentLen, urlContainer)
		return
	} else if statCode == 405 {
		configureAndBindInnerUrl(str_based_url, STATUS_NOTALLOWED, LINK_TYPE_PAGE, contentLen, urlContainer)
		return
	} else if statCode == 418 {
		configureAndBindInnerUrl(str_based_url, STATUS_TEAPOT, LINK_TYPE_PAGE, contentLen, urlContainer)
		return
	} else if statCode == 999 {
		configureAndBindInnerUrl(str_based_url, STATUS_TEAPOT, LINK_TYPE_PAGE, contentLen, urlContainer)
		return
	}
	configureAndBindInnerUrl(str_based_url, STATUS_FAILURE, LINK_TYPE_PAGE, -1, urlContainer)
}

//fmt.Println("Status:", resp.StatusCode)
//fmt.Println("ContentLength:", resp.ContentLength)

func CheckUrls(searchedUrl string, listOfUrls *[]string, urlTree *UrlTreeStruct, progress func(string, float64)) {
	group := new(errgroup.Group)
	group.SetLimit(max_outer_pool)
	if urlTree == nil {
		return
	}
	host := urlTree.Url
	base, err := nurl.Parse(host)
	if err != nil {
		log.Fatal("base fckd ", err)
	}
	lenOfList := float64(len(*listOfUrls))
	group.Go(func() error {
		checkUrl(*base, searchedUrl, urlTree, progress, 0, lenOfList)
		progress(fmt.Sprintf("Checked %s", searchedUrl), float64(0)/lenOfList)
		return nil
	})
	for i, nextUrl := range *listOfUrls {
		nurl := nextUrl
		nextUrlId := i
		group.Go(func() error {
			checkUrl(*base, nurl, urlTree, progress, float64(nextUrlId+1), lenOfList)
			progress(fmt.Sprintf("Checked %s", nurl), float64(nextUrlId+1)/lenOfList)
			return nil
		})
	}
	group.Wait()
}
