package kismetClient

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

type KismetDBClient struct {
	db *sql.DB
	ready bool
}

type KismetDBError string

func (err KismetDBError) Error() string {
	return string(err)
}

func (client *KismetDBClient) SelectFrom(table string, columns []string) (*sql.Rows, error) {
	var query strings.Builder

	query.WriteString("select ")
	if len(columns) == 0 {
		return &sql.Rows{}, KismetDBError("No Columns to select from the table")
	} else {
		for _, column := range columns {
			query.WriteString(fmt.Sprintf("%s ", column))
		}
	}
	query.WriteString(table)

	if rows, err := client.db.Query(query.String()) ; err == nil {
		return rows, nil
	} else {
		return &sql.Rows{}, KismetDBError(fmt.Sprint("DB Query failed: ", err))
	}
}

func NewDBClient(dbFile string) (KismetDBClient, error) {
	var (
		db *sql.DB
	)

	if _, err := os.Stat(dbFile) ; os.IsExist(err) {
		return KismetDBClient{}, KismetDBError(fmt.Sprintf("%s does not exist!", dbFile))
	}

	if newDB, err := sql.Open("sqlite3", dbFile) ; err == nil {
		db = newDB
	} else {
		return KismetDBClient{}, KismetDBError("Failed to create DB connection")
	}

	return KismetDBClient{
		db,
		true,
	}, nil

}

func (dbCli *KismetDBClient) Finish() error {
	dbCli.ready = false
	return dbCli.db.Close()
}
