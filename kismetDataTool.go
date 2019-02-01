package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"kismetDataTool/kismetClient"
	"log"
	"net/url"
	"os"
	"strings"
)

var (
	kismetUrl string
	kismetUsername string
	kismetPassword string
	filterSpec string

	help      bool
	debug     bool
	databaseMode bool
	restMode bool

	dlog      *log.Logger
	ilog      *log.Logger
)

func init() {
	const (
		urlUsage   = "Used to identify the Kismet server"
		usernameUsage = "The username to authenticate to Kismet with"
		passwordUsage = "The password to authenticate to Kismet with"
		filterUsage = "Used to set a filter for either the database or the rest api. " +
			"See /system/tracked_fields.html for more info about filtering in rest mode. " +
			"See the database tables for more info about filtering in database mode. " +
			"Rest filters should be in the format of 'kismet.device.base.macaddr kismet.device.base.phyname'"

		helpUsage  = "Display this help info and exit"
		debugUsage = "Enable debug output"
		dbUsage = "Set database mode"
		restUsage = "Set REST mode"

		urlDefault = "http://127.0.0.1:2501"
		debugDefault = false
	)

	flag.BoolVar(&databaseMode, "db", false, dbUsage)
	flag.BoolVar(&restMode, "rest", false, restUsage)
	flag.StringVar(&kismetUrl, "url", urlDefault, urlUsage)
	flag.BoolVar(&help, "help", false, helpUsage)
	flag.BoolVar(&debug, "verbose", debugDefault, debugUsage)
	flag.StringVar(&kismetUsername, "username", "", usernameUsage)
	flag.StringVar(&kismetPassword, "password", "", passwordUsage)
	flag.StringVar(&filterSpec, "filter", "", filterUsage)
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
	defer dlog.Println("FINISH")

	dlog.Println("Parsing command line options")
	if databaseMode == restMode {
		flag.PrintDefaults()
		ilog.Println(databaseMode, restMode)
		ilog.Println("Please choose either database or rest mode.")
		return
	}

	// TODO: IMPLEMENT
	if databaseMode {
		doDB()
	} else { // REST mode
		// Test the REST parameters
		// TODO: Potentially temporary, Might prompt for creds so users don't have to clear cmd
		//  history to remove saved history file creds
		if kismetUsername == "" || kismetPassword == "" {
			flag.PrintDefaults()
			ilog.Println("You must specify a username and password!")
			return
		}

		if testUrl, err := url.Parse(kismetUrl) ; err == nil {
			if !(testUrl.Scheme == "http") && !(testUrl.Scheme == "https") {
				flag.PrintDefaults()
				ilog.Println("Please enter a valid `http` or `https` url")
				return
			}
		} else {
			flag.PrintDefaults()
			ilog.Println("Please enter a valid `http` or `https` url")
			return
		}

		// Basic check. If they are bad filters, let kismet error out instead of us :D
		if filterSpec == "" {
			flag.PrintDefaults()
			ilog.Println("Please specify filters for rest calls")
			return
		}

		doRest()
	}
}

func doRest() {
	dlog.Println("Creating Kismet client")

	var kClient kismetClient.KismetWebClient

	if newKClient, err := kismetClient.NewWebClient(kismetUrl, kismetUsername, kismetPassword) ; err == nil {
		dlog.Println("Successfully created kismet client")
		kClient = newKClient
	} else {
		ilog.Printf("Failed to create kismet client: %v\n", err)
		return
	}

	// TODO: Remove. Here just to test
	dlog.Println("Auth cookie:", kClient.GetCookie())
	dlog.Println("Lets get some data!")

	filters := strings.Split(filterSpec, " ")

	var jsonResponse []byte

	if reader := kClient.GetDevicesByFilter(filters) ; reader != nil {

		// Going to use ioutil.ReadAll() to read the output pipe into a byte slice
		if result, err := ioutil.ReadAll(reader) ; err == nil {
			jsonResponse = result

			// Now that we have sent the raw filters, we need to simplify them for matching
			// the way kismet want's us to in it's JSON response
			for n, val := range filters {
				if strings.Contains(val, "/") {
					vals := strings.Split(val, "/")
					filters[n] = vals[len(vals) - 1]
				}
			}
		} else {
			ilog.Println("Failed to read result of rest call")
			return
		}
		reader.Close()
	}

	// TODO: REMOVE
	ioutil.WriteFile("log.txt", jsonResponse, 0644)
	if json.Valid(jsonResponse) {
		dlog.Println("Valid JSON response from /devices/summary/devices.json with filters:", filterSpec)
		ilog.Println("Got response from Kismet")
	} else {
		dlog.Println("Invalid JSON response from /devices/summary/devices.json with filters:", filterSpec)
		ilog.Println("Got invalid response from Kismet")
		return
	}

	// Array of maps with string keys and interface{} values (GENERICS! :D)
	var assembledJson []map[string]interface{}

	if err := json.Unmarshal(jsonResponse, &assembledJson) ; err != nil {
		dlog.Println("Decoding error: ", err)
		ilog.Println("Failed to decode JSON response")
		return
	}

	// TODO: Create data mapping technique (kml, csv, etc), delete what's below.

	for n, jMap := range assembledJson {
		dlog.Printf("Response: %d:\n", n)
		for _, filter := range filters {

			jVal := jMap[filter]
			switch jVal.(type) {
			case string:
				dlog.Printf("\t%v: %s string\n", filter, jVal)
			case float64:
				dlog.Printf("\t%v: %g f64\n", filter, jVal)
			case bool:
				dlog.Printf("\t%v: %t bool\n", filter, jVal)
			default:
				dlog.Println("UNHANDLED!")
			}
		}
	}
}

func doDB() {
	ilog.Println("UNIMPLEMENTED!")
}
