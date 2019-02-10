package kismetClient

import "database/sql"

type KismetDBError string

func (err KismetDBError) Error() string {
	return string(err)
}

func (client KismetDBClient) GetRawRows() *sql.Rows {
	return client.rows
}

func (client *KismetDBClient) Finish() error {
	client.ready = false
	client.rows.Close()
	return client.db.Close()
}
