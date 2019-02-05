package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"kismetDataTool/kismetClient"
	"log"
	"net/url"
	"os"
	"strings"
)

var (
	// Flags
	kismetUrl string
	kismetDB string
	filterSpec string

	help      bool
	debug     bool

	// Program vars
	kismetUsername string
	kismetPassword string

	dbMode bool
	restMode bool

	dlog      *log.Logger
	ilog      *log.Logger
)

func init() {
	const (
		urlUsage   = "Used to identify the URL to access the Kismet REST API"
		dbUsage = "Used to identify a local Kismet sqlite3 database file"
		filterUsage = "This flag is used to set a filter for the Kismet REST API if the -restURL " +
			"flag is used, **or** this flag is used to set a filter for a kismet sqlite3 database. " +
			"When using the -restURL flag, filters must be specified space delineated in a single string. " +
			"See /system/tracked_fields.html for a list of fields that this " +
			"program will use to filter requests and results. " +
			"When using the -dbFile flag, filters must be specified by their column names " +
			"in their respective tables. A valid dbFile filter might look like the following: " +
			"`devices/devmac devices/avg_lat` etc. All dbFile filters must specify the same table."

		helpUsage  = "Display this help info and exit"
		debugUsage = "Enable debug output"

		debugDefault = true
	)

	flag.StringVar(&kismetDB, "dbFile", "", dbUsage)
	flag.StringVar(&kismetUrl, "restURL", "", urlUsage)
	flag.StringVar(&filterSpec, "filter", "", filterUsage)

	flag.BoolVar(&help, "help", false, helpUsage)
	flag.BoolVar(&debug, "verbose", debugDefault, debugUsage)

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
		dlog = log.New(ioutil.Discard, "", 0)
	}

	defer dlog.Println("FINISH")

	dlog.Println("Parsing command line options")
	if kismetUrl == kismetDB {
		flag.PrintDefaults()
		ilog.Println("Please choose either database or rest mode.")
		return
	} else if kismetUrl != "" {
		restMode = true
	} else {
		dbMode = true
	}

	if dbMode {
		// TODO: IMPLEMENT

		doDB()
	} else { // REST mode

		// Test the url and filter flags before prompting for username and password
		if testUrl, err := url.Parse(kismetUrl) ; err == nil {
			if !(testUrl.Scheme == "http") || !(testUrl.Scheme == "https") {
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

		// Get kismet username and password
		ilog.Print("Kismet username: ")
		if _, err := fmt.Scanf("%s", &kismetUsername) ; err != nil {
			ilog.Println("Failed to read username")
			return
		}

		ilog.Print("Kismet password: ")
		if _, err := fmt.Scanf("%s", &kismetPassword) ; err != nil {
			ilog.Println("Failed to read password")
			return
		}

		// Test the username and password parameters
		if kismetUsername == "" || kismetPassword == "" {
			flag.PrintDefaults()
			ilog.Println("You must specify a username and password!")
			return
		}

		doRest()
	}
}

func doRest() {
	dlog.Println("Creating Kismet client")

	var kClient kismetClient.KismetRestClient
	if newKClient, err := kismetClient.NewRestClient(kismetUrl, kismetUsername, kismetPassword) ; err == nil {
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
	var (
		dbClient kismetClient.KismetDBClient
		filters []string
	)

	if newClient, err := kismetClient.NewDBClient(kismetDB) ; err == nil {
		dbClient = newClient
		defer dbClient.Finish()
	}

	// get teh filters
	filters = strings.Split(filterSpec, " ")

	var table string
	columns := make([]string, 0)
	for _, v := range filters {
		subFilter := strings.Split(v, "/")
		if len(subFilter) != 2 {
			ilog.Println("Bad DB Filter!")
			os.Exit(1)
		}

		newTable := subFilter[0]
		columns = append(columns, subFilter[1])
		if table == "" {
			table = newTable
		} else if table != newTable {
			ilog.Println("Bad DB Filter!")
			os.Exit(1)
		}
	}

	if rows, err := dbClient.SelectFrom(table, columns) ; err == nil {
		defer rows.Close()

		// TODO: FINISH
	}

	ilog.Println("UNIMPLEMENTED!")
}
