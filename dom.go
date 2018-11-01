package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"os"
	"regexp"
	"strings"
	"time"
)

type PropReader StringMap
type CrawlerConfig map[string]PropReader
type PropCollection StringMap

func isXML(path string) bool {
	return strings.HasSuffix(path, ".xml")
}

func getDOM(link string) *goquery.Document {
	doc, err := goquery.NewDocument(link)
	if err != nil {
		return nil
	}
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

func readProp(doc *goquery.Document, desc PropReader, result StringMap) string {
	node := getNode(doc, desc)
	var value string = ""

	var concatWith string = desc["concatWith"]
	if concatWith == "" {
		concatWith = " "
	}

	switch desc["prop"] {
	case "Text":
		value = node.Text()

	case "Attr":
		attr, _ := node.Attr(desc["Attr"])
		value = attr

	case "Re":
		r := regexp.MustCompile(desc["Re"])
		match := r.FindStringSubmatch(result[desc["useValue"]])
		if len(match) > 1 {
			value = match[1]
		} else {
			value = ""
		}
	}
	return trim(value, concatWith)
}

func getCrawlerOutput(name string) string {
	return "./result/crawler/"+name+".crawler.txt"
}

func getParserOutput(name string) string {
	return "./result/parser/"+name+".parser.txt"
}

func removeRawValues(item StringMap) StringMap {
	var formatted StringMap = make(StringMap)
	for key, value := range item {
		if !strings.HasPrefix(key, "raw_") {
			formatted[key] = value
		}
	}
	return formatted
}

func crawl(config map[string]CrawlerConfig, name string, bootOptions map[string]string) {
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
		cat_links    []string = grepLinks(doc, crawlerOptions["menu"]["selector"])
		links        []string
		unique_links map[string]bool = make(map[string]bool)
	)

	_normalizeLink := func(link string) string {
		return normalizeLink(link, start)
	}

	for i := 0; i < len(cat_links); i++ {
		cat_link := cat_links[i]
		link := _normalizeLink(cat_link)
		doc = getDOM(link)
		products := grepLinks(doc, crawlerOptions["item"]["selector"])
		links = append(links, mapArray(products, _normalizeLink)...)

		if unique_links[link] != true {
			unique_links[link] = true
			pages := doc.Find(crawlerOptions["pagination"]["selector"])
			if pages.Length() > 1 {
				last_page, _ := getInt(pages.Last().Text())
				if last_page > 1 {
					var page_links []string
					pages.Each(func(i int, node *goquery.Selection) {
						href, _ := node.Attr("href")
						page_links = append(page_links, href)
						link = _normalizeLink(href)
						unique_links[link] = true
						fmt.Println(link)
					})
					cat_links = append(cat_links, page_links[1:]...)
				}
			}
		}
		
		fmt.Println("Crawled " + link)
		writeLines(links, getCrawlerOutput(name))
	}
}

func parse(config map[string]CrawlerConfig, name string, bootOptions StringMap) PropCollection {
  parser := config["parser"]
  var (
  	result []StringMap
  	links []string
  )

	readLines(getCrawlerOutput(name), func(link string) {
		links = append(links, link)
	})

	channel_length := len(links)
	poolsize, _ := getInt(bootOptions["poolsize"])
	sleeptime, _ := getInt(bootOptions["sleeptime"])
	jobs := make(chan WorkerJob, channel_length)
  results := make(chan StringMap, channel_length)
  for w := 1; w <= poolsize; w++ {
    go worker(w, jobs, results)
  }

  for _, link := range links {
  	(func(link string) {
			jobs <- func() StringMap {
				if sleeptime > 0 {
					time.Sleep(time.Second*time.Duration(sleeptime))
				}				
	  		doc := getDOM(link)
	  		if doc == nil {
	  			fmt.Println("Failed to load " + link)
	  			return nil
	  		}

				item := make(StringMap)
				item["url"] = link
				for name, prop := range parser {
					item[name] = readProp(doc, prop, item)
				}
				fmt.Println("Parsed: " + link)
				return removeRawValues(item)
			}  
  	})(link)
  }
  close(jobs)
	
  for a := 0; a < channel_length; a++ {
    result = append(result, <- results)
    if len(result) % 10 == 0 {
			writeJson(result, getParserOutput(name))
			fmt.Println("Saved count: ", len(result))
    }
  }

	writeJson(result, getParserOutput(name))

	return nil
}