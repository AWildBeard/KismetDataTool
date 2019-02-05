package kismetClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type KismetRestClient struct {
	url string
	ready bool
	authCookie http.Cookie
}

type KismetRequestError string

func (err KismetRequestError) Error() string {
	return string(err)
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

// Returns
func (client *KismetRestClient) GetDevicesByFilter(filters []string) *io.PipeReader {
	var (
		jsonReader io.Reader
		jsonObj = map[string][]string{
			"fields": filters,
		}
		jsonLen int
	)

	if jsonBytes, err := json.Marshal(jsonObj); err == nil {
		// Success!

		jsonLen = len(jsonBytes) + 5 // json=
		jsonReader = io.MultiReader(strings.NewReader("json="), bytes.NewReader(jsonBytes))
	} else {
		return nil
	}

	if newRequest, err := http.NewRequest("POST", client.url + customQueryPath, jsonReader) ; err == nil {
		// Success
		request = newRequest
	} else {
		return nil
	}

	request.AddCookie(&client.authCookie)
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Charset", "utf-8")
	request.Header.Add("Content-Length", strconv.Itoa(jsonLen))

	if newResponse, err := httpClient.Do(request) ; err == nil {
		reader, writer := io.Pipe()

		go func () {
			defer writer.Close()
			defer newResponse.Body.Close()
			io.Copy(writer, newResponse.Body)
		}()

		return reader
	} else {
		return nil
	}


}

// Returns a Kismet Web Client ready to make REST API requests. This method will make Web API requests in order
// to retrieve the authentication token
func NewRestClient(url, username, password string) (KismetRestClient, error) {
	var (
		authCookie http.Cookie
	)

	if newRequest, err := http.NewRequest("GET", url + authPath, strings.NewReader("")) ; err == nil {
		// Creating the request was successful, so lets just store the request for when we use it :D
		request = newRequest
		defer request.Body.Close()
	} else { // Creating the request was not successful
		return KismetRestClient{}, KismetRequestError(fmt.Sprintf("Failed to create request to %s.\n" +
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
		return KismetRestClient{}, KismetRequestError("Failed to authenticate to Kismet.\n" +
			"Perhaps your username/ password combination is incorrect?")
	}

	// Verify the kismet client
	kismetClient := KismetRestClient{
		url,
		true,
		authCookie,
	}

	if kismetClient.ValidConnection() {
		return kismetClient, nil
	} else {
		return KismetRestClient{}, KismetRequestError("Failed to validate authentication cookie.")
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

func (client *KismetRestClient) Finish() error {
	client.ready = false
	return nil
}

func (client *KismetRestClient) IsValid() bool {
	return client.ready
}

