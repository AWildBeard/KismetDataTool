package kismetClient

import "net/http"

// Helpers for the Rest Client
func (client *KismetRestClient) GetCookie() *http.Cookie {
	return &client.authCookie
}
