package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"os"
	"strings"
	"sync"
)

func _check(err error) {
	if err != nil {
		panic(err)
	}
}

func parseUrl(url string) {
	fmt.Println("request: " + url)
	doc, err := goquery.NewDocument(url)
	_check(err)

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		fmt.Println(link)
	})
}

func isXML(path string) bool {
	return strings.HasSuffix(path, ".xml")
}

func getDOM(link string, local bool) (doc *goquery.Document, err error) {
	if local {
		return localDOM(link)
	} else {
		return goquery.NewDocument(link)
	}
}

func localDOM(path string) (doc *goquery.Document, err error) {
	var f *os.File
	f, err = os.Open(path)
	defer f.Close()
	return goquery.NewDocumentFromReader(f)
}

func parseSitemap(link string, local bool, accumulator []string) []string {
	doc, err := getDOM(link, local)
	_check(err)

	doc.Find("loc").Each(func(i int, node *goquery.Selection) {
		href := node.Text()
		if isXML(href) {
			parseSitemap(href, false, accumulator)
		} else {
			parseHTML(href)
			//accumulator = append(accumulator, href)
		}
	})

	return accumulator
}

var i bool = false

func parseHTML(link string) {
	if i {
		return
	}
	i = true

	doc, err := getDOM(link, false)
	_check(err)
	fmt.Printf("Title = %s, src = %s", doc.Find("h1").Text(), link)
}

func main() {
	var wg sync.WaitGroup

	for _, url := range os.Args[1:] {
		wg.Add(1)

		go func(url string) {
			defer wg.Done()
			parseUrl(url)
		}(url)
	}

	wg.Wait()

	var products []string
	products = parseSitemap("auchan/sitemap_index.xml", true, products)
	fmt.Println(products)
}
