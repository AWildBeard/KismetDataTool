package kismetClient

// Helpers for the Rest Client

type KismetRestError string

func (err KismetRestError) Error() string {
	return string(err)
}

func (client *KismetRestClient) Finish() error {
	if request != nil {
		request.Body.Close()
	}
	client.Ready = false
	return nil
}
