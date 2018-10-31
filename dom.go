package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func isXML(path string) bool {
	return strings.HasSuffix(path, ".xml")
}

func getDOM(link string) *goquery.Document {
	doc, err := goquery.NewDocument(link)
	_check(err)
	return doc
}

func localDOM(path string) (doc *goquery.Document, err error) {
	var f *os.File
	f, err = os.Open(path)
	defer f.Close()
	return goquery.NewDocumentFromReader(f)
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

func grepLinks(doc *goquery.Document, selector string) []string {
	var links map[string]bool = make(map[string]bool)
	doc.Find(selector).Each(func(i int, node *goquery.Selection) {
		link, _ := node.Attr("href")
		links[link] = true
	})
	return getKeys(links)
}

func normalizeLink(link, origin string) string {
	if strings.HasPrefix(link, "./") {
		parts := strings.Split(origin, "/")
		return strings.Join(parts[:len(parts)-1], "/") +
			strings.Replace(link, "./", "/", -1)
	}
	if strings.HasPrefix(link, "../") {
		parts := strings.Split(origin, "/")
		return strings.Join(parts[:len(parts)-2], "/") +
			strings.Replace(link, "../", "/", -1)
	}
	if !strings.HasPrefix(link, "http") {
		return origin + link
	}
	return link
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

func parse(link string, reader CrawlerConfig) PropCollection {
	doc := getDOM(link)
	result := make(PropCollection)
	result["url"] = link
	for name, prop := range reader {
		result[name] = readProp(doc, prop, result)
	}

	return result
}

func crawl(config map[string]CrawlerConfig, name string) {
	crawlerOptions := config["crawler"]
	var (
		origin string = crawlerOptions["root"]["origin"]
		start  string = crawlerOptions["root"]["start"]
	)
	if start == "" {
		start = origin
	}

	doc := getDOM(start)

	var (
		cat_links    []string
		links        []string
		unique_links map[string]bool = make(map[string]bool)
	)

	cat_links = grepLinks(doc, crawlerOptions["menu"]["selector"])

	for i := 0; i < len(cat_links); i++ {
		cat_link := cat_links[i]

		link := normalizeLink(cat_link, start)
		doc = getDOM(link)
		products := grepLinks(doc, crawlerOptions["item"]["selector"])

		links = append(links, mapArray(products, func(s string) string {
			return normalizeLink(s, start)
		})...)

		if unique_links[link] != true {
			unique_links[link] = true
			pages := doc.Find(crawlerOptions["pagination"]["selector"])
			if pages.Length() > 1 {
				last_page, _ := strconv.Atoi(trim(pages.Last().Text(), ""))

				if last_page > 1 {
					var page_links []string
					pages.Each(func(i int, node *goquery.Selection) {
						href, _ := node.Attr("href")
						page_links = append(page_links, href)

						link = normalizeLink(href, start)
						unique_links[link] = true

						fmt.Println(link)
					})
					cat_links = append(cat_links, page_links[1:]...)
				}
			}
		}

		writeLines(links, "./result/"+name+".crawler.txt")
		fmt.Println("Crawled " + link)
	}

	/*var result []PropCollection
	  var parsed PropCollection
	  var i int = 0
	  for _, link := range links {
	    if i >= 50 {
	      break
	    }
	    if strings.TrimSpace(link) == "" {
	      continue
	    }

	    parsed = parseHTML(normalizeLink(link, start), config["parser"])
	    if parsed["description"] == "" {
	      fmt.Printf("No description on %s\n", link)
	    } else {
	      i = i + 1
	      result = append(result, parsed)
	    }

	    fmt.Println("Parsed " + link)
	  }

	  data, _ := json.Marshal(result)
	  var result_json []string
	  result_json = append(result_json, string(data))
	  writeLines(result_json, name+"-results.json")*/
}
