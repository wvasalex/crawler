package main

import (
	"flag"
	"fmt"
)

var (
	app    = flag.String("app", "", "Application name. Crawler on parser?")
	config = flag.String("config", "", "Config file name from ./config dir")
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

func main() {
	flag.Parse()

	//processSitemap("auchan/sitemap_index.xml", "auchan/products.txt")

	if *config == "" {
		fmt.Println("Parser name required")
		return
	}

	if *app == "crawler" {
		crawl(readConfig(*config), *config)
	}

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
}
