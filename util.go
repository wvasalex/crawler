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
	"strconv"
)

type StringMap map[string]string
type LineReaderHandler func(string)
type WorkerJob func() StringMap

func _check(err error) {
	if err != nil {
		panic(err)
	}
}

var space_re = regexp.MustCompile(`\s\s+`)
var newline_re = regexp.MustCompile(`\n\n+`)
func trim(s, replaceWith string) string {
	return space_re.ReplaceAllString(
		newline_re.ReplaceAllString(strings.TrimSpace(s), "/"),
		replaceWith)
}

var digital_re = regexp.MustCompile("([0-9]+)")
func getInt(s string) (int, error) {
	match := digital_re.FindStringSubmatch(s)
	return strconv.Atoi(match[1])
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

func writeJson(raw []StringMap, path string) error {
	data, _ := json.Marshal(raw)
  var result_json []string
  result_json = append(result_json, string(data))
  return writeLines(result_json, path)
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

func stringifyJSON(raw StringMap) string {
	data, _ := json.Marshal(raw)
	return string(data)
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

func worker(id int, jobs <-chan WorkerJob, results chan<- StringMap) {
	for job := range jobs {
		result := job()
		if result != nil {
			results <- result
		}
	}
}