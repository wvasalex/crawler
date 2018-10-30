package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

var (
	parserName = flag.String("name", "", "Parser name")
)

type LineReaderHandler func(string)
type PropReader map[string]string
type CrawlerConfig map[string]PropReader
type PropCollection map[string]string

func _check(err error) {
	if err != nil {
		panic(err)
	}
}

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

	doc := getDOM(link)
	doc.Find("loc").Each(func(i int, node *goquery.Selection) {
		href := node.Text()
		if isXML(href) {
			accumulator = append(accumulator, parseSitemap(href, false)...)
		} else {
			accumulator = append(accumulator, href)
		}
	})

	return accumulator
}

func parseHTML(link string, reader CrawlerConfig) PropCollection {
	doc := getDOM(link)
	result := make(PropCollection)
	result["url"] = link
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

func getKeys(m map[string]bool) []string {
	v := make([]string, 0, len(m))
	for key, _ := range m {
		v = append(v, key)
	}
	return v
}

func processSitemap(source, output string) []string {
	var products []string
	products = parseSitemap(source, true)
	if output != "" {
		writeLines(products, output)
	}
	return products
}

func readConfig(name string) map[string]CrawlerConfig {
	b, err := ioutil.ReadFile("config/" + name + ".json")
	_check(err)
	var config map[string]CrawlerConfig
	err = json.Unmarshal(b, &config)
	_check(err)
	return config
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
	if !strings.HasPrefix(link, origin) {
		return origin + link
	}
	return link
}

func crawl(config map[string]CrawlerConfig, name string) {
	crawlerOptions := config["crawler"]
	var origin string = crawlerOptions["root"]["origin"]
	doc := getDOM(origin)
	var cat_links []string = grepLinks(doc, crawlerOptions["menu"]["selector"])
	var links []string
	for _, cat_link := range cat_links {
		page_dom := getDOM(normalizeLink(cat_link, origin))
		links = append(links, grepLinks(page_dom, crawlerOptions["item"]["selector"])...)

		fmt.Println("Crawled " + cat_link)
	}

	//doc, _ := getDOM("https://www.auchan.ru/pokupki/kosmetika/uhod-za-volosami/stajling.html", false)
	//var pages []string = grepLinks(doc, crawlerOptions["pagination"]["selector"])

	var result []PropCollection
	var parsed PropCollection
	var i int = 0
	for _, link := range links {
		if i >= 50 {
			break
		}
		if strings.TrimSpace(link) == "" {
			continue
		}

		parsed = parseHTML(normalizeLink(link, origin), config["parser"])
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
	writeLines(result_json, name+"-results.json")
}

func main() {
	flag.Parse()

	//processSitemap("auchan/sitemap_index.xml", "auchan/products.txt")

	if *parserName == "" {
		fmt.Println("Parser name required")
		return
	}

	var config map[string]CrawlerConfig = readConfig(*parserName)
	crawl(config, *parserName)

	/*var wg sync.WaitGroup
	readLines("products.txt", func(link string) {
		if i > 50 {
			return
		}

		info := parseHTML(link)
		if info["description"] == "" {
			fmt.Printf("No description on %s\n", link)
		} else {
			i = i + 1
			products = append(products, info)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			info := parseHTML(link)
			if info["description"] == "" {
				fmt.Printf("No description on %s\n", link)
			} else {
				i = i + 1
				products = append(products, info)
			}
		}()
	})*/

	//wg.Wait()

	/*data, _ := json.Marshal(products)
	var result []string
	result = append(result, string(data))
	writeLines(result, "result.json")*/
	//parseHTML("https://www.auchan.ru/pokupki/cosmia-kr-lica-i-tel-uvlazh-50m.html")
}
