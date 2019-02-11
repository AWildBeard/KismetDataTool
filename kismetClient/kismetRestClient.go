package kismetClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type KismetRestClient struct {
	url string
	ready bool
	authCookie http.Cookie
	filters []string
}

var (
	httpClient http.Client
	request *http.Request
)

const (
	authPath = "/session/check_login"
	authCheckPath = "/session/check_session"
	customQueryPath = "/devices/summary/devices.json"
	kismetAuthCookieName = "KISMET"
)

// Make the kismet request for the devices with their filters and return a function generator
// that returns single device information.
func (client *KismetRestClient) Elements() (func() (DataElement, error), error) {
	var (
		// Needed to create a JSON string
		jsonRequestObj = map[string][]string{
			"fields": client.filters,
		}
		// Sentinel error return value
		badFunc = func() (DataElement, error) { return DataElement{}, KismetRestError("Failed to create generator") }
		// Trimmed down filters to match the expected kismet response
		responseFilters = make([]string, len(client.filters))
		// The JSON response from kismet converted into a native go object
		assembledJson []map[string]interface{}
	)

	if !client.ready {
		return badFunc, KismetRestError("Client is not ready")
	}

	for n, val := range client.filters {
		if strings.Contains(val, "/") {
			vals := strings.Split(val, "/")
			responseFilters[n] = vals[len(vals) - 1]
		} else {
			responseFilters[n] = val
		}
	}

	// Create the JSON string
	if jsonBytes, err := json.Marshal(jsonRequestObj); err == nil {
		jsonLen := len(jsonBytes) + 5 // json=
		jsonReader := io.MultiReader(strings.NewReader("json="), bytes.NewReader(jsonBytes))

		// Create the HTTP Request
		if newRequest, err := http.NewRequest("POST", client.url + customQueryPath, jsonReader) ; err == nil {
			// Success
			request = newRequest

			// Add relevant parts to the HTTP Request
			request.AddCookie(&client.authCookie)
			request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			request.Header.Add("Charset", "utf-8")
			request.Header.Add("Content-Length", strconv.Itoa(jsonLen))

			// Send the request and handle the response
			if newResponse, err := httpClient.Do(request) ; err == nil { // Returns JSON doc
				if jsonResponse, err := ioutil.ReadAll(newResponse.Body) ; err == nil { // Read JSON doc
					if !json.Valid(jsonResponse) { // Validate JSON doc
						return badFunc, KismetRestError(fmt.Sprint("Got invalid JSON from Kismet with filters:", client.filters))
					}

					// Dissasemble the JSON and store the dissasembled obj in assembledJson
					if err := json.Unmarshal(jsonResponse, &assembledJson) ; err != nil {
						return badFunc, KismetRestError(fmt.Sprint("Failed to decode JSON response:", err))
					}
				}
				newResponse.Body.Close()
			} else {
				return badFunc, KismetRestError(fmt.Sprint("Error handling HTTP request", err))
			}
			request.Body.Close()
		} else {
			return badFunc, KismetRestError(fmt.Sprint("Failed to create HTTP request:", err))
		}
	} else {
		return badFunc, KismetRestError("Failed to create JSON request")
	}

	offset := 0
	numDevices := len(assembledJson)
	return func() (DataElement, error) {
		element := DataElement{}

		if offset >= numDevices {
			return element, KismetRestError("No more devices left")
		}

		device := assembledJson[offset]

		for i := 0 ; i < 2 ; i++ {
			switch device[responseFilters[i]].(type) {
			case float64:
			default:
				return element, KismetRestError(
					fmt.Sprintf("Improper first element %v Please see the help page for more info.",
						device[responseFilters[i]]))
			}
		}

		element.Lat = float64(device[responseFilters[0]].(float64))
		element.Lon = float64(device[responseFilters[1]].(float64))

		switch id := device[responseFilters[2]]; id.(type) {
		case string:
			element.ID = id.(string)
		case int:
			element.ID = string(id.(int))
		default:
			return element, KismetRestError(
				fmt.Sprint("Invalid ID field from parsed data:", device[responseFilters[2]]))
		}

		element.HasData = true

		numFilters := len(responseFilters)
		if numFilters - 3 > 0 {
			element.extraData = true
			extraData := make([]interface{}, numFilters - 3)
			for n, filter := range responseFilters[3:] {
				extraData[n] = device[filter]
			}
		}

		offset++
		return element, nil
	}, nil
}

// Returns a Kismet Web Client ready to make REST API requests. This method will make Web API requests in order
// to retrieve the authentication token
func NewRestClient(url, username, password string, filters []string) (KismetRestClient, error) {
	var (
		authCookie http.Cookie
	)

	if newRequest, err := http.NewRequest("GET", url + authPath, strings.NewReader("")) ; err == nil {
		// Creating the request was successful, so lets just store the request for when we use it :D
		request = newRequest
		defer request.Body.Close()
	} else { // Creating the request was not successful
		return KismetRestClient{}, KismetRestError(fmt.Sprintf("Failed to create request to %s.\n" +
			"Perhaps you forgot to add http:// to the beginning of the url?", url))
	}

	request.SetBasicAuth(username, password)

	if newResponse, err := httpClient.Do(request) ; err == nil && newResponse.StatusCode == 200 {
		// Performing the request was successful
		for _, cookie := range newResponse.Cookies() {
			if cookie.Name == kismetAuthCookieName {
				authCookie = *cookie // Copy :D
			}
		}
		defer newResponse.Body.Close()
	} // Don't check for this error (err) case.
	// If the kismet cookie isn't set, we check below which ends up covering this error case

	if authCookie.Name != kismetAuthCookieName {
		return KismetRestClient{}, KismetRestError("Failed to authenticate to Kismet.\n" +
			"Perhaps your username/ password combination is incorrect?")
	}

	// Verify the kismet client
	kismetClient := KismetRestClient{
		url,
		true,
		authCookie,
		filters,
	}

	if kismetClient.ValidConnection() {
		return kismetClient, nil
	} else {
		return KismetRestClient{}, KismetRestError("Failed to validate authentication cookie.")
	}
}

// Tests for a valid connection. The implementation tests the Kismet authentication cookie.
func (client *KismetRestClient) ValidConnection() bool {
	if newRequest, err := http.NewRequest("GET", client.url + authCheckPath, strings.NewReader("")) ; err == nil {
		request = newRequest
		defer request.Body.Close()
	} else {
		return false
	}

	request.AddCookie(&client.authCookie) // DOH

	if response, err := httpClient.Do(request) ; err == nil && response.StatusCode == 200 {
		return true
	}

	return false
}
