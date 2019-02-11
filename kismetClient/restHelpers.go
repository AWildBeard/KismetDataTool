package kismetClient

import "net/http"

// Helpers for the Rest Client

type KismetRestError string

func (err KismetRestError) Error() string {
	return string(err)
}

func (client *KismetRestClient) GetCookie() *http.Cookie {
	return &client.authCookie
}

func (client *KismetRestClient) Finish() error {
	request.Body.Close()
	client.ready = false
	return nil
}

func (client *KismetRestClient) IsReady() bool {
	return client.ready
}
