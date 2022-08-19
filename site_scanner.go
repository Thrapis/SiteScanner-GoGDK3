package main

import (
	"fmt"
	"log"
	"net/http"
	nurl "net/url"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/PuerkitoBio/goquery"
	nlzurl "github.com/sekimura/go-normalize-url"
)

type CounterUnit struct {
	Count int
}

func (cu *CounterUnit) NextValue() (result int) {
	result = cu.Count
	cu.Count++
	return
}

func (cu *CounterUnit) GeneratedCount() int {
	return cu.Count
}

var mtx sync.Mutex

func SafeGetByValue(pages *map[string]int, value int) (string, bool) {
	mtx.Lock()
	defer mtx.Unlock()

	for k, v := range *pages {
		if v == value {
			return k, true
		}
	}
	return "", false
}

func Normalize(url string) (string, error) {
	norm_url, err := nlzurl.Normalize(url)
	if err != nil {
		return "", err
	}
	norm_url += "/"

	norm_url, err = nurl.QueryUnescape(norm_url)
	if err != nil {
		return "", err
	}

	return norm_url, nil
}

func StartScan(norm_url string, progress func(string, float64)) *[]string {

	client := &http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	var pages = map[string]int{}
	var counter = &CounterUnit{Count: 0}
	var currentIndex = counter.NextValue()
	pages[norm_url] = currentIndex

	group := new(errgroup.Group)
	group.SetLimit(max_pool)
	freeChan := make(chan struct{}, max_pool)
	defer close(freeChan)

	var scanCounter = &CounterUnit{Count: 0}
	var next = scanCounter.NextValue()

	freeChan <- struct{}{}
	group.Go(func() error {
		scanNextPage(client, norm_url, &pages, next, counter, progress)
		<-freeChan
		return nil
	})
	for counter.GeneratedCount() != scanCounter.GeneratedCount() || len(freeChan) > 0 {
		if counter.GeneratedCount() != scanCounter.GeneratedCount() {
			freeChan <- struct{}{}
			next_one := scanCounter.NextValue()
			group.Go(func() error {
				scanNextPage(client, norm_url, &pages, next_one, counter, progress)
				<-freeChan
				return nil
			})
		}
	}

	group.Wait()

	pages_arr := make([]string, 0, len(pages)-1)
	for k, _ := range pages {
		if k != norm_url {
			pages_arr = append(pages_arr, k)
		}
	}

	sort.Slice(pages_arr, func(i, j int) bool {
		l1, l2 := len(pages_arr[i]), len(pages_arr[j])
		if l1 != l2 {
			return l1 < l2
		}
		return pages_arr[i] < pages_arr[j]
	})

	return &pages_arr
}

func scanNextPage(client *http.Client, host string, pages *map[string]int, index int, counter *CounterUnit, progress func(string, float64)) {

	get_url, ok := SafeGetByValue(pages, index)
	if !ok {
		return
	}
	norm_url, err := nlzurl.Normalize(get_url)
	if err != nil {
		return
	}
	norm_url += "/"

	//fmt.Println("Check", norm_url)

	base, err := nurl.Parse(host)
	if err != nil {
		log.Fatal("base fff ", err)
	}

	resp, err := client.Get(norm_url)
	if err != nil {
		return
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}

	as := doc.Find("a")

	as.Each(func(i int, a *goquery.Selection) {
		if href, ok := a.Attr("href"); ok {
			//clear := href
			href_url, err := base.Parse(href)
			if err == nil {
				href_url.RawQuery = ""
				href_url.Fragment = ""
				href, err = nurl.QueryUnescape(href_url.String())
				if err != nil {
					log.Println("Query unescape fail!!!", err)
				}
				if strings.Contains(href, host) {
					//fmt.Println("Clear/Dirt", clear, "/", href)
					fileExtension := filepath.Ext(href)
					if len(fileExtension) == 0 {
						addAllCombinatons(host, href, pages, counter)
					}
				}
			}
		}
	})
	progress(fmt.Sprintf("Process page %s", norm_url), float64(index)/float64(counter.GeneratedCount()))
}

func addAllCombinatons(host, href string, pages *map[string]int, counter *CounterUnit) {

	href, err := Normalize(href)
	if err != nil {
		log.Println("normalize err ", err)
	}
	href = strings.ReplaceAll(href, "\\", "/")

	nugget_href := strings.ReplaceAll(href, host, "")

	splittes := strings.SplitAfter(nugget_href, "/")
	buf_href := host

	for _, split := range splittes {
		if strings.Contains(split, ".") {
			continue
		}
		buf_href = fmt.Sprintf("%s%s", buf_href, split)
		norm_url, err := Normalize(buf_href)
		if err != nil {
			log.Println("normalize err ", err)
		}
		mtx.Lock()
		if _, ok := (*pages)[norm_url]; !ok {
			(*pages)[norm_url] = counter.NextValue()
		}
		mtx.Unlock()
	}
}
