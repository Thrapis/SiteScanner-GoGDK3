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
		//fmt.Println("Uts not found!")
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
			fmt.Println("Too long to wait")
			return
		default:
			uts.Status = STATUS_FAILURE
			return
		}
	}
	statCode := resp.StatusCode
	//fmt.Println("Code of", url, "is", statCode)
	if statCode == 200 {
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			uts.Status = STATUS_PROBLEM
			fmt.Println("Can not read body")
			return
		}
		//fmt.Println("Size of", url, "document is", doc.Length())
		imgs := doc.Find("img")
		as := doc.Find("a")
		asCount := float64(as.Length())
		partCoeff := 1 / asCount / count
		group := new(errgroup.Group)
		group.SetLimit(max_inner_pool)
		uts.InnerUrls = make([]UrlStruct, 0)
		imgs.Each(func(i int, a *goquery.Selection) {
			if href, ok := a.Attr("src"); ok {
				if err == nil {
					group.Go(func() error {
						checkInnerUrl(base, href, uts, INTENT_SRC)
						progress(fmt.Sprintf("Checked image %s", href), (partCoeff*float64(i)+index)/count)
						return nil
					})
				} else {
					fmt.Println("have no href? ", err)
				}
			}
		})
		as.Each(func(i int, a *goquery.Selection) {
			if href, ok := a.Attr("href"); ok {
				if err == nil {
					group.Go(func() error {
						checkInnerUrl(base, href, uts, INTENT_HREF)
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
	fmt.Println("Stat code", statCode)
	uts.Status = statCode
}

func configureAndBindInnerUrl(url string, status, linkType, intent int, sourceSize int64, urlContainer *UrlTreeStruct) {
	urlElement := NewUrlStruct(url)
	urlElement.Status = status
	urlElement.LinkType = linkType
	urlElement.SourceSize = sourceSize
	urlElement.Intent = intent
	urlContainer.AppendInnerUrl(urlElement)
}

func checkInnerUrl(base nurl.URL, url string, urlContainer *UrlTreeStruct, intent int) {
	based_url, err := base.Parse(url)
	if err != nil {
		fmt.Println("try base failed:", url)
		configureAndBindInnerUrl(url, STATUS_FAILURE, LINK_TYPE_PAGE, intent, -1, urlContainer)
		return
	}
	str_based_url := based_url.String()
	//fmt.Println("Sceme of", url, "is", part_url.Scheme)
	switch based_url.Scheme {
	case SCHEME_MAILTO:
		configureAndBindInnerUrl(str_based_url, STATUS_PROBLEM, LINK_TYPE_MAILTO, intent, -1, urlContainer)
		return
	case SCHEME_TEL:
		configureAndBindInnerUrl(str_based_url, STATUS_PROBLEM, LINK_TYPE_TEL, intent, -1, urlContainer)
		return
	case SCHEME_CALLTO:
		configureAndBindInnerUrl(str_based_url, STATUS_PROBLEM, LINK_TYPE_CALLTO, intent, -1, urlContainer)
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
			configureAndBindInnerUrl(str_based_url, STATUS_LONGWAIT, LINK_TYPE_PAGE, intent, -1, urlContainer)
			return
		default:
			configureAndBindInnerUrl(str_based_url, STATUS_FAILURE, LINK_TYPE_PAGE, intent, -1, urlContainer)
			return
		}
	}
	statCode := resp.StatusCode
	contentLen := resp.ContentLength
	//fmt.Println("Code of inner", url, "is", statCode)
	if statCode == 200 {
		configureAndBindInnerUrl(str_based_url, STATUS_SUCCESS, LINK_TYPE_PAGE, intent, contentLen, urlContainer)
		return
	} else if statCode >= 300 && statCode <= 308 {
		newUrl, err := resp.Location()
		if err != nil {
			checkInnerUrl(base, newUrl.String(), urlTree, intent)
			return
		}
	}
	fmt.Println("Stat code", statCode)
	configureAndBindInnerUrl(str_based_url, statCode, LINK_TYPE_PAGE, intent, contentLen, urlContainer)
}

//fmt.Println("Status:", resp.StatusCode)
//fmt.Println("ContentLength:", resp.ContentLength)

func InitCheckUrls(searchedUrl string, listOfUrls *[]string, urlTree *UrlTreeStruct, progress func(string, float64)) {
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

func InitCheckUrl(searchedUrl string, selectedUrl *UrlTreeStruct, progress func(string, float64)) {
	if selectedUrl == nil {
		return
	}
	host := searchedUrl
	base, err := nurl.Parse(host)
	if err != nil {
		log.Fatal("base fckd ", err)
	}
	checkUrl(*base, selectedUrl.Url, selectedUrl, progress, 0, 0.8)
	progress(fmt.Sprintf("Checked %s", searchedUrl), 0.95)
}

func InitCheckUrlDeep(searchedUrl string, selectedUrl *UrlTreeStruct, progress func(string, float64)) {
	if selectedUrl == nil {
		return
	}
	host := searchedUrl
	base, err := nurl.Parse(host)
	if err != nil {
		log.Fatal("base fckd ", err)
	}
	checkUrl(*base, selectedUrl.Url, selectedUrl, progress, 0, 0.4)

	max := max_outer_pool + selectedUrl.Deep()
	limitChan := make(chan struct{}, max)
	group := new(errgroup.Group)
	group.SetLimit(max_outer_pool)

	for _, child := range selectedUrl.Childs {
		nurl := child.Url
		chld := child
		group.Go(func() error {
			nextDeep(searchedUrl, chld, limitChan, progress)
			progress(fmt.Sprintf("Checked %s", nurl), 0.9)
			return nil
		})
	}
	group.Wait()
	progress(fmt.Sprintf("Checked %s", searchedUrl), 0.95)
}

func nextDeep(searchedUrl string, selectedUrl *UrlTreeStruct, limitChan chan struct{}, progress func(string, float64)) {
	limitChan <- struct{}{}
	if selectedUrl == nil {
		return
	}
	host := searchedUrl
	base, err := nurl.Parse(host)
	if err != nil {
		log.Fatal("base fckd ", err)
	}
	checkUrl(*base, selectedUrl.Url, selectedUrl, progress, 0, 0.4)
	group := new(errgroup.Group)
	group.SetLimit(max_outer_pool)

	for _, child := range selectedUrl.Childs {
		nurl := child.Url
		chld := child
		group.Go(func() error {
			InitCheckUrlDeep(searchedUrl, chld, progress)
			progress(fmt.Sprintf("Checked %s", nurl), 0.9)
			return nil
		})
	}
	group.Wait()
	progress(fmt.Sprintf("Checked %s", searchedUrl), 0.95)
	<-limitChan
}
