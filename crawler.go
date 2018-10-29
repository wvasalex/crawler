package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"os"
	"regexp"
	"strings"
	"sync"
)

type LineReaderHandler func(string)

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

func writeLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

func readLines(path string, handler LineReaderHandler) error {
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		handler(sc.Text())
	}
	return sc.Err()
}

func parseSitemap(link string, local bool) []string {
	var accumulator []string

	doc, err := getDOM(link, local)
	_check(err)

	doc.Find("loc").Each(func(i int, node *goquery.Selection) {
		href := node.Text()
		if isXML(href) {
			accumulator = append(accumulator, parseSitemap(href, false)...)
		} else {
			accumulator = append(accumulator, href)
			//parseHTML(href)
		}

	})

	return accumulator
}

type PropReader map[string]string
type PropCollection map[string]string

func parseHTML(link string) PropCollection {
	doc, err := getDOM(link, false)
	_check(err)

	reader := make(map[string]PropReader)
	reader["title"] = PropReader{"selector": "h1", "prop": "Text"}
	reader["url"] = PropReader{"selector": "meta[property='og:url']", "prop": "Attr", "Attr": "content"}
	reader["category"] = PropReader{"selector": "li.breadcrumbs__item", "prop": "Text"}
	reader["price"] = PropReader{"selector": "div.current-price", "filter": "Last", "prop": "Text"}
	reader["description"] = PropReader{"selector": "ul.prcard__feat-list", "filter": "Last", "prop": "Text"}
	reader["code"] = PropReader{"useValue": "description", "re": "Артикул:/(?P<code>\\w+)", "prop": "Re"}

	result := make(PropCollection)
	for name, prop := range reader {
		result[name] = readProp(doc, prop, result)
	}

	return result
}

func getNode(doc *goquery.Document, desc PropReader) *goquery.Selection {
	if desc["selector"] == "" {
		return nil
	}

	node := doc.Find(desc["selector"])

	switch desc["filter"] {
	case "Last":
		node = node.Last()

	case "First":
		node = node.First()
	}

	return node
}

func readProp(doc *goquery.Document, desc PropReader, result PropCollection) string {
	node := getNode(doc, desc)
	var value string = ""

	switch desc["prop"] {
	case "Text":
		value = node.Text()

	case "Attr":
		attr, _ := node.Attr(desc["Attr"])
		value = attr

	case "Re":
		r := regexp.MustCompile(desc["re"])
		match := r.FindStringSubmatch(result[desc["useValue"]])
		if len(match) >= 1 {
			value = match[1]
		} else {
			value = ""
		}
	}

	r := regexp.MustCompile("\n+")
	value = r.ReplaceAllString(strings.TrimSpace(value), "/")

	return value
}

func getValues(m PropCollection) []string {
	v := make([]string, 0, len(m))

	for _, value := range m {
		v = append(v, value)
	}
	return v
}

func main() {
	/*var wg sync.WaitGroup

	for _, url := range os.Args[1:] {
		wg.Add(1)

		go func(url string) {
			defer wg.Done()
			parseUrl(url)
		}(url)
	}

	wg.Wait()*/

	/*var products []string
	products = parseSitemap("auchan/sitemap_index.xml", true)
	fmt.Println(products)

	writeLines(products, "products.txt")*/

	//var products []PropCollection

	var i int = 0

	file, _ := os.Create("result.csv")
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = '\t'
	defer writer.Flush()

	var wg sync.WaitGroup
	readLines("products.txt", func(link string) {
		if i > 10 {
			return
		}

		i = i + 1
		fmt.Println(link)

		wg.Add(1)
		go func() {
			defer wg.Done()
			info := parseHTML(link)
			writer.Write(getValues(info))
		}()

	})

	wg.Wait()

	//parseHTML("https://www.auchan.ru/pokupki/cosmia-kr-lica-i-tel-uvlazh-50m.html")
}
