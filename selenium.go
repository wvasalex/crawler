package main

import (
	"fmt"
	"github.com/tebeka/selenium"
	"time"
)

func GetWithRuntime(url string) (string) {
	const (
		// These paths will be different on your system.
		seleniumPath    = "../src/github.com/tebeka/selenium/vendor/selenium-server-standalone-3.14.0.jar"
		geckoDriverPath = "../src/github.com/tebeka/selenium/vendor/geckodriver-v0.23.0-linux64"
		port            = 8888
	)
	opts := []selenium.ServiceOption{
		//selenium.StartFrameBuffer(),           // Start an X frame buffer for the browser to run in.
		selenium.GeckoDriver(geckoDriverPath), // Specify the path to GeckoDriver in order to use Firefox.
		//selenium.Output(os.Stderr),            // Output debug information to STDERR.
	}
	//selenium.SetDebug(true)
	service, err := selenium.NewSeleniumService(seleniumPath, port, opts...)
	if err != nil {
		panic(err) // panic is used only as an example and is not otherwise recommended.
	}
	defer service.Stop()

	// Connect to the WebDriver instance running locally.
	caps := selenium.Capabilities{"browserName": "firefox"}
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
	if err != nil {
		panic(err)
	}
	defer wd.Quit()

	// Navigate to the simple playground interface.
	if err := wd.Get(url); err != nil {
		panic(err)
	}

	time.Sleep(time.Second*20)

	v, a := wd.FindElement(selenium.ByCSSSelector, "#names")
	fmt.Println(a,v)

	//html, err := wd.ExecuteScriptRaw("return document.documentElement.outerHTML", nil)
	//html, _ := elem.GetAttribute("innerHTML")
	html, _ := wd.PageSource()
	return html
}