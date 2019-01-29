package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

var (
	url string
	help bool
	debug bool
)

func init() {
	const (
		urlUsage   = "Used to identify the Kismet server"
		helpUsage  = "Display this help info and exit"
		debugUsage = "Enable debug output"

		urlDefault = "http://127.0.0.1:2501"
		debugDefault = true
	)

	flag.StringVar(&url, "url", urlDefault, urlUsage)
	flag.StringVar(&url, "u", urlDefault, urlUsage+ " (shorthand)")
	flag.BoolVar(&help, "h", false, helpUsage)
	flag.BoolVar(&debug, "v", debugDefault, debugUsage)
}

func main() {
	flag.Parse()
	if help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	log("Testing connectivity to Kismet instance at", url)

	if response, err := http.Get(url + "/datasource/all_sources.json") ; err == nil {
		dlogf("Got Response: %v, using protocol: %v", response.StatusCode, response.Proto)
	} else {
		logf("Failed to connect to %v :(\nExiting", url)
		os.Exit(1)
	}
}

func log(dat string, v ...interface{}) {
	if v != nil {
		fmt.Println(dat, v)
	} else {
		fmt.Println(dat)
	}
}

func dlog(dat string, v ...interface{}) {
	if debug {
		if v != nil {
			fmt.Println(dat, v)
		} else {
			fmt.Println(dat)
		}
	}
}

func logf(dat string, v ...interface{}) {
	if v != nil {
		fmt.Printf(dat, v)
	} else {
		fmt.Printf(dat)
	}
}

func dlogf(dat string, v ...interface{}) {
	if debug {
		if v != nil {
			fmt.Printf(dat, v)
		} else {
			fmt.Printf(dat)
		}
	}
}
