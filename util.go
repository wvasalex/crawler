package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

var linebreak_re = regexp.MustCompile("\n+")

func trim(s, replaceWith string) string {
	return linebreak_re.ReplaceAllString(strings.TrimSpace(s), replaceWith)
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

func processSitemap(source, output string) []string {
	var products []string
	products = parseSitemap(source, true)
	if output != "" {
		writeLines(products, output)
	}
	return products
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

func readConfig(name string) map[string]CrawlerConfig {
	b, err := ioutil.ReadFile("config/" + name + ".json")
	_check(err)
	var config map[string]CrawlerConfig
	err = json.Unmarshal(b, &config)
	_check(err)
	return config
}

func mapArray(vs []string, f func(string) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}
