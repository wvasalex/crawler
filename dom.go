package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"os"
	"regexp"
	"strings"
	"time"
	"golang.org/x/text/encoding/charmap"
)

type PropReader StringMap
type CrawlerConfig map[string]PropReader
type PropCollection StringMap

func decode(ba string) string {
	return ba;
	dec := charmap.Windows1251.NewDecoder()
	out, _ := dec.Bytes([]byte(ba))
	return string(out)
}

func isXML(path string) bool {
	return strings.HasSuffix(path, ".xml")
}

func getDOM(link string) *goquery.Document {
	doc, err := goquery.NewDocument(link)
	if err != nil {
		panic(err)
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

	if desc["filter"] != "" {
		switch desc["filter"] {
		case "Last":
			node = node.Last()

		case "First":
			node = node.First()

		default:
			var len int = node.Length()
			if len > 0 {
				var slices []string = strings.Split(desc["filter"], ":")
				start, _ := getInt(slices[0])
				end, _ := getInt(slices[1])
				if end == 0 || end > len {
					end = len
				}
				node = node.Slice(start, end)
			}			
		}
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

func normalizeLink(link, origin, base string) string {
	if strings.HasPrefix(link, "./") {
		parts := strings.Split(base, "/")
		return strings.Join(parts[:len(parts)-1], "/") +
			strings.Replace(link, "./", "/", -1)
	}
	if strings.HasPrefix(link, "../") {
		parts := strings.Split(base, "/")
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
		value = strings.Join(node.Map(func(i int, item *goquery.Selection) string {
			return item.Text()
		}), concatWith)

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

	case "Table":
		keys := node.Find(desc["header"])
		values := node.Find(desc["value"])

		var table StringMap = make(StringMap)
		keys.Each(func(i int, node *goquery.Selection) {
			var key string = trim(node.Text(), "")
			table[decode(key)] = decode(trim(values.Eq(i).Text(), ""))
		})
		for key, value := range desc {
			if strings.HasPrefix(key, "&") {
				result[strings.Replace(key, "&", "", 1)] = table[value]
			}
		}

		value = stringifyJSON(table)
		return value
	}

	return decode(trim(value, concatWith))
}

func getCrawlerOutput(name string) string {
	return "./result/crawler/"+name+".crawler.txt"
}

func getParserOutput(name string) string {
	return "./result/parser/"+name+".parser.json"
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
		debug bool = bootOptions["debug"] != ""
		origin string = crawlerOptions["root"]["origin"]
		start  string = crawlerOptions["root"]["start"]
	)
	if start == "" {
		start = origin
	}

	doc := getDOM(start)

	if doc == nil {
		fmt.Println("DOC is broken!",start)
		return
	}

	var (
		cat_links    []string = grepLinks(doc, crawlerOptions["menu"]["selector"])
		links        []string
		unique_links map[string]bool = make(map[string]bool)
	)

	if debug && len(cat_links) > 3 {
		cat_links = cat_links[:3]
	}

	_normalizeLink := func(link string) string {
		return normalizeLink(link, origin, start)
	}

	for i := 0; i < len(cat_links); i++ {
		cat_link := cat_links[i]
		link := _normalizeLink(cat_link)
		doc = getDOM(link)
		products := grepLinks(doc, crawlerOptions["item"]["selector"])
		links = append(links, mapArray(products, _normalizeLink)...)

		if unique_links[link] != true {
			unique_links[link] = true
			pages := getNode(doc, crawlerOptions["pagination"])		
			if pages.Length() > 1 {
				last_page, _ := getInt(pages.Last().Text())
				if last_page > 1 {
					fmt.Println("last page is",pages.Length(),last_page)
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
  	debug bool = bootOptions["debug"] != ""
  	result []StringMap
  	links []string
  )

	readLines(getCrawlerOutput(name), func(link string) {
		links = append(links, link)
	})
	if debug {
		links = links[:10]
	}

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
	//.map(a => { a.category=a.category.replace(/([а-яё])([А-ЯЁ])/g, '$1/$2');  return a})

	return nil
}
