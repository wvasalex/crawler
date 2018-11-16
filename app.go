package main

import (
	"flag"
	"fmt"
)

var (
	app       = flag.String("app", "", "Application name. Crawler | Parser")
	config    = flag.String("config", "", "Config file name from ./config dir")
	poolsize  = flag.String("poolsize", "50", "Workers pool size")
	sleeptime = flag.String("sleeptime", "2", "Worker sleep time before start, sec.")
	debug     = flag.String("debug", "", "Is app in debug mode")
)

func main() {
	flag.Parse()

	if *app == "olegon" {
		GetWithRuntime("https://barcodes.olegon.ru/index.php?c=8009307013098")
		//writeFile([]byte(html), "./result/output/8009307013098")
		
		return
	}

	if *config == "" {
		fmt.Println("Parser name required")
		return
	}

	var readerConfig map[string]CrawlerConfig = readConfig(*config)
	var bootConfig StringMap
	bootConfig = make(StringMap)
	bootConfig["poolsize"] = *poolsize
	bootConfig["sleeptime"] = *sleeptime
	bootConfig["debug"] = *debug

	if *app == "crawler" {
		crawl(readerConfig, *config, bootConfig)
	}
	if *app == "parser" {
		parse(readerConfig, *config, bootConfig)
	}
}
