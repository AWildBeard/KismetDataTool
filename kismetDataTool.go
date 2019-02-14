package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/AWildBeard/kismetDataTool/kismetClient"
	"io"
	"io/ioutil"
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
	output string

	help      bool
	debug     bool

	// Program vars
	kismetUsername string
	kismetPassword string

	appendMode bool
	dbMode bool
	restMode bool

	outputFunc func(reader kismetClient.DataLineReader) error

	outputWriter io.Writer

	dlog      *log.Logger
	ilog      *log.Logger
)

func init() {
	const (
		urlUsage   = "Used to identify the URL to access the Kismet REST API ``\n"
		dbUsage = "Used to identify a local Kismet sqlite3 database file ``\n"
		filterUsage = "This flag is used to set a filter for the Kismet REST API if the\n" +
			"-restURL flag is used, **or** this flag is used to set a filter\n" +
			"for a kismet sqlite3 database if the -dbFile flag is used. This\n" +
			"flag must have some specification of the Latitude, Longitude,\n" +
			"and Device ID from the kismet data source. This is mandatory\n" +
			"as this program is a device focused data tool. It is not\n" +
			"planned to support other areas of kismet's data sources.\n" +
			"The argument for this flag should be a string ``\n\n" + // Wierdness here to customize output
			"When using the -restURL flag, filters must be specified space\n" +
			"delineated in a single string. See /system/tracked_fields.html\n" +
			"on your kismet server for a list of fields that this program\n" +
			"will use to filter requests and results for the REST endpoint.\n\n" +
			"When using the -dbFile flag, filters must be specified by their\n" +
			"column names in their respective tables. A valid dbFile filter\n" +
			"might look like the following: `devices/devmac devices/avg_lat`\n" +
			"etc. All dbFile filters must specify the same table.\n"
		outputUsage = "``Used to select the destination output for the data gathered by\n" + // Weirdness here to customize output
			"this program. To write the data to STDOUT (in a csv-like\n" +
			"manner) the argument should be `-`. File format specifications\n" +
			"are determined from the file extension of the argument to this\n" +
			"flag. For example, if you would like to output a csv, you would\n" +
			"use the flag `-output out.csv`. The default is to output in a\n" +
			"csv-like manner to stdout. The default is to write to STDOUT.\n" +
			"The supported file formats are: csv, kml\n"

		appendUsage = "Do not print headers for CSV mode. (append)\n"
		helpUsage  = "Display this help info and exit\n"
		debugUsage = "Enable debug output (written to STDERR)\n"

		debugDefault = false
	)

	flag.StringVar(&kismetDB, "dbFile", "", dbUsage)
	flag.StringVar(&kismetUrl, "restUrl", "", urlUsage)
	flag.StringVar(&filterSpec, "filter", "", filterUsage)
	flag.StringVar(&output, "output", "", outputUsage)

	flag.BoolVar(&help, "help", false, helpUsage)
	flag.BoolVar(&debug, "verbose", debugDefault, debugUsage)
	flag.BoolVar(&appendMode, "append", false, appendUsage)

	flag.Usage = usage
}

func main() {
	output = "-" // Set a default that doesn't change the help pages output
	flag.Parse()
	if help {
		usage()
		return
	}

	ilog = log.New(os.Stdout, "", 0)
	defer fmt.Fprintln(os.Stderr, "Exiting. Have a good day! (っ◕‿◕)っ")

	if debug {
		dlog = log.New(os.Stderr, "DEBUG: ", log.Ltime)
	} else {
		dlog = log.New(ioutil.Discard, "", 0)
	}

	defer dlog.Println("FINISH")

	dlog.Println("Parsing command line options")
	if kismetUrl == kismetDB {
		usage()
		ilog.Println("Please choose either database or rest mode.")
		return
	} else if kismetUrl != "" {
		restMode = true
	} else {
		dbMode = true
	}

	if output == "-" {
		outputWriter = os.Stdout
		outputFunc = writeCsv
	} else if strings.Contains(output, ".csv") {
		var mode int
		if appendMode { mode = os.O_APPEND } else { mode = os.O_CREATE }
		if newFile, err := os.OpenFile(output, mode, 0666) ; err == nil {
			outputWriter = newFile
		} else {
			dlog.Printf("Failed to open file %v: %v", output, err)
			ilog.Println("Could not open selected file")
		}
		outputFunc = writeCsv
	} else if strings.Contains(output, ".kml") {
		var mode int
		if appendMode { mode = os.O_APPEND } else { mode = os.O_CREATE }
		if newFile, err := os.OpenFile(output, mode, 0666) ; err == nil {
			outputWriter = newFile
		} else {
			dlog.Printf("Failed to open file %v: %v", output, err)
			ilog.Println("Could not open selected file")
		}
		outputFunc = writeKml
	} else {
		dlog.Println("Invalid output format specified:", output)
		ilog.Println("Please choose a supported output format. See the help page for more info.")
		return
	}

	if dbMode { // DB mode
		var (
			table string
			columns []string
		)

		dlog.Println("Parsing db filters")
		columns = make([]string, 0)
		for _, v := range strings.Split(filterSpec, " ") {
			subFilter := strings.Split(v, "/")
			if len(subFilter) != 2 {
				ilog.Println("Bad DB Filter:", v)
				return
			}

			newTable := subFilter[0]
			columns = append(columns, subFilter[1])
			if table == "" {
				table = newTable
			} else if table != newTable {
				ilog.Println("Bad DB Filter:", v)
				return
			}
		}
		dlog.Println("Using table:", table)
		dlog.Println("Using columns:", columns)
		dlog.Println("Successfully parsed DB filters")

		dlog.Println("Running database command")
		doDB(table, columns)
	} else { // REST mode
		// Test the url and filter flags before prompting for username and password
		if testUrl, err := url.Parse(kismetUrl) ; err == nil {
			if !(testUrl.Scheme == "http") && !(testUrl.Scheme == "https") {
				usage()
				dlog.Println("URL does not appear to have http or https protocol:", testUrl.Scheme)
				ilog.Println("Please enter a valid `http` or `https` url")
				return
			}
		} else {
			usage()
			dlog.Println("Failed to create url:", err)
			ilog.Println("Please enter a valid `http` or `https` url")
			return
		}
		dlog.Println("Using Kismet URL:", kismetUrl)

		// Basic check. If they are bad filters, let kismet error out instead of us :D
		if filterSpec == "" {
			usage()
			ilog.Println("Please specify filters for rest calls")
			return
		}
		dlog.Println("Using filters:", filterSpec)

		// Get kismet username and password
		fmt.Print("Kismet username: ")
		if _, err := fmt.Scanf("%s", &kismetUsername) ; err != nil {
			ilog.Println("Failed to read username")
			return
		}

		fmt.Print("Kismet password: ")
		if _, err := fmt.Scanf("%s", &kismetPassword) ; err != nil {
			ilog.Println("Failed to read password")
			return
		}

		// Test the username and password parameters
		if kismetUsername == "" || kismetPassword == "" {
			usage()
			ilog.Println("You must specify a username and password!")
			return
		}

		dlog.Println("Successfully parsed required options for kismet REST client")

		dlog.Println("Running REST command")
		doRest(strings.Split(filterSpec, " "))
	}
}

func doRest(restFilters []string) {
	var (
		kClient kismetClient.KismetRestClient
	)

	dlog.Println("Creating Kismet client")
	// Get a client to the Kismet REST api
	if newKClient, err := kismetClient.NewRestClient(kismetUrl, kismetUsername, kismetPassword, restFilters) ; err == nil {
		dlog.Println("Created kismet client")
		kClient = newKClient
		defer kClient.Finish()
	} else {
		dlog.Println("Failed to create kismet client: ", err)
		ilog.Println("Failed to connect to kismet")
		return
	}

	// Write the elements
	if err := outputFunc(&kClient) ; err != nil {
		dlog.Println("Error writing output:", err)
	}
}

func doDB(table string, columns []string) {
	var (
		dbClient kismetClient.KismetDBClient
	)

	dlog.Println("Creating Kismet client")
	// Get a client to the Kismet sqlite3 database
	if newClient, err := kismetClient.NewDBClient(kismetDB, table, columns) ; err == nil {
		dlog.Println("Created Kismet client")
		dbClient = newClient
		defer dbClient.Finish() // Cleanup
	} else {
		dlog.Println("Failed to create a DB Connection:", err)
		ilog.Println("Failed to read database")
		return
	}

	// Write the elements
	// So apparently referencing a type that implements a supertype makes it compatible with that supertype
	if err := outputFunc(&dbClient) ; err != nil {
		dlog.Println("Error writing output:", err)
	}
}

func writeCsv(client kismetClient.DataLineReader) error {
	var (
		clientGenerator func () (kismetClient.DataElement, error)
		byteBuffer *bytes.Buffer
	)

	{
		templateBuffer := make([]byte, 0, 4096)
		byteBuffer = bytes.NewBuffer(templateBuffer)
	}

	dlog.Println("Creating element generator")
	if newGenerator, err := client.Elements() ; err == nil {
		clientGenerator = newGenerator
	} else {
		dlog.Println("Failed to create element generator")
		return err
	}

	// Print header
	if !appendMode { // Open a new scope
		dlog.Println("Writing csv header")
		headers := client.ElementHeaders()
		headerLen := len(headers)
		for n, v := range headers {
			if byteBuffer.Len() >= byteBuffer.Cap() {
				if _, err := outputWriter.Write(byteBuffer.Bytes()); err != nil {
					return err
				}
			}

			if n == headerLen-1 {
				byteBuffer.WriteString(fmt.Sprintf("%v", v))
			} else {
				byteBuffer.WriteString(fmt.Sprintf("%v,", v))
			}

		}
		byteBuffer.WriteByte('\n') // Newline
	}

	// Print elements
	dlog.Println("Writing elements")
	for elem, err := clientGenerator() ; err == nil && elem.HasData ; elem, err = clientGenerator() {
		var (
			stringBuilder strings.Builder
			outputString string
		)
		if _, err := fmt.Fprintf(&stringBuilder, "%v,%v,%v", elem.Lat, elem.Lon, elem.ID) ; err != nil {
			return err
		}

		if elem.HasExtraData() {
			if _, err := fmt.Fprint(&stringBuilder, ",") ; err != nil {
				return err
			}

			extraData := elem.GetExtraData()
			extraDataLen := len(*extraData)
			for n, v := range *extraData {
				if n == extraDataLen - 1 {
					if _, err := fmt.Fprintf(&stringBuilder, "%v", v) ; err != nil {
						return err
					}
				} else {
					if _, err := fmt.Fprintf(&stringBuilder, "%v,", v) ; err != nil {
						return err
					}
				}
			}
		}

		if _, err := fmt.Fprintln(&stringBuilder); err != nil {
			return err
		}

		// We've built the string now
		outputString = stringBuilder.String()
		stringBuilder.Reset()

		if byteBuffer.Len() + len(outputString) >= byteBuffer.Cap() {
			if _, err := outputWriter.Write(byteBuffer.Bytes()) ; err != nil {
				return err
			}
			byteBuffer.Reset()
		}

		byteBuffer.WriteString(outputString)
	}

	return nil
}

func writeKml(client kismetClient.DataLineReader) error {
	return kismetClient.KismetRestError("UNIMPLEMENTED")
}

func usage() {
	fmt.Fprint(os.Stderr, `NAME
  kismetDataTool

DESCRIPTION
  This program allows for a limited extraction of device-related
  data from different kismet sources such as a kismet sqlite3 
  database, or kismet's REST API endpoint. By program paradigm,
  any query to a kismet source must include a method to retrieve
  the latitude of the device in the first filter position,
  a method to retrieve the longitude of the device in the second
  filter position, and an appropriate ID for the device in the
  third filter position. This ID can be an integer from kismet
  or a string (such as devmac in the kismet sqlite3 database)
  See below for a synopsis of command line arguments

`)
	flag.PrintDefaults()
	fmt.Fprint(os.Stderr, `EXAMPLES

  Retrieve the latitude, longitude, and MAC of a device from 
  the REST API endpoint and write the output to STDOUT

	kismetDataTool -restUrl 'http://127.0.0.1:2501' \
	-filter '\
	kismet.device.base.location/kismet.common.location.\
	avg_loc/kismet.common.location.lat \
	kismet.device.base.location/kismet.common.location.\
	avg_loc/kismet.common.location.lon \
	kismet.device.base.macaddr' \

  Same as above but with the kismet sqlite3 database

	kismetDataTool -dbFile kismet-x.kismet \
	-filter 'devices/avg_lat devices/avg_lon devices/devmac'

AUTHOR
  Michael Mitchell

  Using libraries written by:
	mattn,
	twpayne

`)
}
