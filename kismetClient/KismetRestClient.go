package kismetClient

import (
	"fmt"
	"net/http"
	"strings"
)

type KismetWebClient struct {
	url string
	ready bool
	authCookie http.Cookie
}

type KismetRequestError string

func (err KismetRequestError) Error() string {
	return string(err)
}

const (
	authPath = "/session/check_login"
	authCheckPath = "/session/check_session"
	customQueryPath = "/devices/summary/devices.json"
	kismetAuthCookieName = "KISMET"
)

// Returns a Kismet Web Client ready to make REST API requests. This method will make Web API requests in order
// to retrieve the authentication token
func NewWebClient(url, username, password string) (KismetWebClient, error) {
	httpClient := http.Client{}

	var (
		request *http.Request
		authCookie http.Cookie
	)

	if newRequest, err := http.NewRequest("GET", url + authPath, strings.NewReader("")) ; err == nil {
		// Creating the request was successful, so lets just store the request for when we use it :D
		request = newRequest
		defer request.Body.Close()
	} else { // Creating the request was not successful
		return KismetWebClient{}, KismetRequestError(fmt.Sprintf("Failed to create request to %s.\n" +
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
	} // Don't check for this error (err) case.
	// If the kismet cookie isn't set, we check below which ends up covering this error case

	// Finalize the request. Not concerned with the error

	if authCookie.Name != kismetAuthCookieName {
		return KismetWebClient{}, KismetRequestError("Failed to authenticate to Kismet.\n" +
			"Perhaps your username/ password combination is incorrect?")
	}

	// Verify the kismet client
	kismetClient := KismetWebClient{
		url,
		true,
		authCookie,
	}

	if kismetClient.ValidConnection() {
		return kismetClient, nil
	} else {
		return KismetWebClient{}, KismetRequestError("Failed to validate authentication cookie.")
	}
}

// Tests for a valid connection. The implementation tests the Kismet authentication cookie.
func (client *KismetWebClient) ValidConnection() bool {
	httpClient := http.Client{}

	var request *http.Request

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

