package main

import (
	"KismetDataTool/kismetClient"
	"flag"
	"io"
	"log"
	"net/url"
	"os"
)

var (
	kismetUrl string
	kismetUsername string
	kismetPassword string

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

	dlog.Println("Parsing command line options")
	if databaseMode == restMode {
		flag.PrintDefaults()
		ilog.Println("Please choose either database or rest mode.")
		return
	}

	// TODO: IMPLEMENT
	if databaseMode {
		ilog.Println("UNIMPLEMENTED!")
		return
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
	}

	var kClient kismetClient.KismetWebClient
	dlog.Println("Creating Kismet client")
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

	if reader := kClient.GetDevicesByFilter([]string {"kismet.device.base.macaddr","kismet.device.base.phyname",}) ; reader != nil {
		// Going to use ioutil.ReadAll() to read the output pipe into a byte slice
		io.Copy(os.Stdout, reader)
		reader.Close()
	}
	ilog.Println()
}
