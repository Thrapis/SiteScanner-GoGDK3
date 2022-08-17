package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/PuerkitoBio/goquery"
	nlzurl "github.com/sekimura/go-normalize-url"
)

func StartScan(url string, progress func(float64)) {
	norm_url, _ := nlzurl.Normalize(url)
	fmt.Println(norm_url)

	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}
	resp, err := client.Get(norm_url)
	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	as := doc.Find("a")

	as.Each(func(i int, a *goquery.Selection) {
		if href, ok := a.Attr("href"); ok {
			fmt.Printf("Href %d: %s\n", i, href)
		} else {
			log.Println("NOT FOUND")
		}
		progress(float64(i) / float64(as.Length()))
	})
}
