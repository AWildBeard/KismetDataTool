package main

import (
	"flag"
	"io"
	"log"
	"net/http"
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
		debugDefault = false
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
		return
	}

	if debug {
		dlog = log.New(os.Stderr, "DEBUG: ", log.Ltime)
	} else {
		if writer, err := os.Open(os.DevNull) ; err == nil {
			dlog = log.New(writer, "", 0)
		} else {
			ilog.Println("Critical logging failure: Can't create null output for debug output")
			return
		}
	}

	/*
	 * TODO: Potentially temporary, Might prompt for creds so users don't have to clear cmd history
	 * to remove saved history file creds
	 */
	if kismetUsername == "" || kismetPassword == "" {
		flag.PrintDefaults()
		ilog.Println("You must specify a username and password!")
		return
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
		ilog.Println("Failed to establish Kismet connection")
		return
	} else {
		request = newRequest
	}

	request.SetBasicAuth(kismetUsername, kismetPassword)

	if newResponse, err := httpClient.Do(request) ; err != nil {
		dlog.Printf("Failed to finalize auth request. Response: %v\nError: %v\n", newResponse, err)
		ilog.Println("Failed to establish Kismet connection")
		return
	} else {
		ilog.Println("Connected to Kismet")
		response = newResponse
	} // Done with request
	request.Body.Close()

	if response.StatusCode != 200 {
		ilog.Println("Failed to authenticate to Kismet")
		return
	} else {
		ilog.Println("Authenticated to Kismet")
	}

	dlog.Println("Cookies:")

	// Looking for KISMET auth cookie
	for _, cookie := range response.Cookies() {
		dlog.Println(cookie)
		if cookie.Name == "KISMET" {
			dlog.Println("Found KISMET auth cookie")
			kismetCookie = *cookie
		}
	} // Done with response
	response.Body.Close()

	if kismetCookie.Name != "KISMET" {
		dlog.Println("Failed to find KISMET auth cookie")
		ilog.Println("Failed to connect to Kismet")
		return
	}

	ilog.Println("Testing KISMET cookie")

	if newRequest, err := http.NewRequest("GET", kismetUrl + "/session/check_session", strings.NewReader("")); err != nil {
		dlog.Printf("Failed to create connection. Request: %v\nError: %v\n", newRequest, err)
		ilog.Println("Failed to establish Kismet connection")
		return
	} else {
		request = newRequest
	}

	request.AddCookie(&kismetCookie)

	if newResponse, err := httpClient.Do(request) ; err != nil {
		dlog.Printf("Failed to finalize auth request. Response: %v\nError: %v\n", newResponse, err)
		ilog.Println("Failed to establish Kismet connection")
		return
	} else {
		response = newResponse
	} // Done with Request
	request.Body.Close()

	if response.StatusCode != 200 {
		ilog.Println("Validation of KISMET cookie failed!")
		return
	} else {
		ilog.Println("Validated KISMET cookie")
	} // Done with response
	response.Body.Close()

	ilog.Println("Attempting to get Device data")

	if newRequest, err := http.NewRequest("GET", kismetUrl + "/devices/all_devices.ekjson", strings.NewReader("")) ; err != nil {
		dlog.Println("Failed to create request for devices")
		ilog.Println("Failed to connect to kismet")
		return
	} else {
		request = newRequest
	}

	// Customize request
	request.AddCookie(&kismetCookie)

	if newResponse, err := httpClient.Do(request) ; err != nil {
		ilog.Println("Failed to connect to kismet")
		return
	} else {
		response = newResponse
	} // Done with request
	request.Body.Close()

	if response.StatusCode != 200 {
		ilog.Println("Failed to get device data!")
		dlog.Printf("Request: %v\n", request)
		dlog.Printf("Response: %v\n", response)
		return
	} else {
		ilog.Println("Devices: ")
		if file, err := os.Create("log.txt") ; err == nil {
			io.Copy(file, response.Body)
		}
	} // Done with response
	response.Body.Close()
}
