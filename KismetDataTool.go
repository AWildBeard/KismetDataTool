package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	kismetUrl string
	help      bool
	debug     bool
	kismetUsername string
	kismetPassword string

	dlog      *log.Logger
	ilog      *log.Logger
	kismetCookie http.Cookie
)

func init() {
	const (
		urlUsage   = "Used to identify the Kismet server"
		helpUsage  = "Display this help info and exit"
		debugUsage = "Enable debug output"
		usernameUsage = "The username to authenticate to Kismet with"
		passwordUsage = "The password to authenticate to Kismet with"

		urlDefault = "http://127.0.0.1:2501"
		debugDefault = true
	)

	flag.StringVar(&kismetUrl, "url", urlDefault, urlUsage)
	flag.BoolVar(&help, "help", false, helpUsage)
	flag.BoolVar(&debug, "verbose", debugDefault, debugUsage)
	flag.StringVar(&kismetUsername, "username", "", usernameUsage)
	flag.StringVar(&kismetPassword, "password", "", passwordUsage)
}

func main() {
	ilog = log.New(os.Stdout, "", 0)
	defer ilog.Println("Exiting. Have a good day! (っ◕‿◕)っ")

	flag.Parse()
	if help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if debug {
		dlog = log.New(os.Stderr, "DEBUG: ", log.Ltime)
	}

	/*
	 * TODO: Potentially temporary, Might prompt for creds so users don't have to clear cmd history
	 * to remove saved history file creds
	 */
	if kismetUsername == "" || kismetPassword == "" {
		flag.PrintDefaults()
		ilog.Println("You must specify a username and password!")
		os.Exit(1)
	}

	ilog.Println("Testing connectivity to Kismet instance at", kismetUrl)
	// Really we're just going to try to authenticate. No sense in continuing if we can't for this POC

	httpClient := http.Client{}

	var (
		request *http.Request
		response *http.Response
	)

	if newRequest, err := http.NewRequest("GET", kismetUrl + "/session/check_login", strings.NewReader("")) ; err != nil {
		dlog.Printf("Failed to create connection. Request: %v\nError: %v\n", newRequest, err)
		ilog.Println("Failed to establish Kismet connection\nExiting..")
		os.Exit(1)
	} else {
		request = newRequest
	}

	request.SetBasicAuth(kismetUsername, kismetPassword)

	if newResponse, err := httpClient.Do(request) ; err != nil {
		dlog.Printf("Failed to finalize auth request. Response: %v\nError: %v\n", newResponse, err)
		ilog.Println("Failed to establish Kismet connection\nExiting..")
		os.Exit(1)
	} else {
		response = newResponse
	}

	// Better safe than sorry
	request.Body.Close()
	response.Body.Close()

	dlog.Printf("Good Response: %v\n", response)
	dlog.Println("Cookies:")

	// Looking for KISMET auth cookie
	for _, cookie := range response.Cookies() {
		dlog.Println(cookie)
		if cookie.Name == "KISMET" {
			dlog.Println("Found KISMET auth cookie")
			kismetCookie = *cookie
		}
	}

	if kismetCookie.Name != "KISMET" {
		dlog.Println("Failed to find KISMET auth cookie")
		ilog.Println("Failed to connect to Kismet\nExiting..")
		os.Exit(0)
	} else {
		ilog.Println("Connected to Kismet")
	}

	return

	// Nothing below here will run for now

	stringBuilder := strings.Builder{}
	jsonEncoder := json.NewEncoder(&stringBuilder)

	if err := jsonEncoder.Encode(map[string][]string {
		"fields": {"kismet.device.base.macaddr","kismet.device.base.phyname",},
	}) ; err != nil {
		ilog.Println("Failed to encode json payload")
		os.Exit(1)
	}

	values := url.Values{}
	values.Add("json", stringBuilder.String())
	dlog.Printf("POST PAYLOAD:\njson=%s\n", values.Get("json"))

	if response, err := http.PostForm(kismetUrl+ "/devices/all_devices.ekjson", values) ; err == nil {
		//dlog.Printf("Got Response: %d, using protocol: %s", response.StatusCode, response.Proto)
		dlog.Printf("Response: %v\n", response)
		io.Copy(os.Stdout, response.Body)
	} else {
		ilog.Printf("Failed to connect to %v :(\nExiting", kismetUrl)
		os.Exit(1)
	}
}
