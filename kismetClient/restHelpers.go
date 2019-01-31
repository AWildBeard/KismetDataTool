package kismetClient

import "net/http"

// Helpers for the Rest Client
func (client *KismetWebClient) GetCookie() *http.Cookie {
	return &client.authCookie
}
