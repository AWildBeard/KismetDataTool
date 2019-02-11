package kismetClient

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"strings"
)

type KismetDBClient struct {
	db *sql.DB
	rows *sql.Rows

	table string
	columns []string

	ready bool
}

// When calling Elements(), the DB Client automatically runs the prepared query
// that was created by calling NewDBClient(). After running the query (which might
// cause Elements() to error out) Elements() returns a generator that can be used
// to retrieve single rows from the previously ran query. For example; if the user
// is running a devices query on a Kismet DB, this would return unique elements
// for each device in the Kismet DB
func (client *KismetDBClient) Elements() (func() (DataElement, error), error) {
	numFilters := len(client.columns)
	rowContent := make([]interface{}, numFilters)

	badFunc := func () (DataElement, error) { return DataElement{}, KismetDBError("Generator not Initialized") }

	if err := client.runQuery() ; err == nil {
		if columnTypes, err := client.rows.ColumnTypes(); err == nil {
			for i, v := range columnTypes {
				theType := v.DatabaseTypeName()
				fmt.Printf("%v: %T\n", theType, theType)
				switch theType {
				case "TEXT":
					var newVal string
					rowContent[i] = &newVal
				case "INT":
					var newVal int
					rowContent[i] = &newVal
				default:
					return badFunc, KismetDBError("Unhandled DB Type from database query. Please only use INT and TEXT columns")
				}
			}
		}

		return func() (DataElement, error) {
			returnElement := DataElement{}

			if client.rows.Next() {
				// Returns elements one row at a time
				if err := client.rows.Scan(rowContent...) ; err != nil {
					return returnElement, KismetDBError("Failed to parse database!")
				}

				for i := 0 ; i < 2 ; i++ {
					switch rowContent[i].(type) {
					case *int:
					default:
						return returnElement, KismetDBError("Lat or Lon field not an int!")
					}
				}

				returnElement.Lat = float64(*rowContent[0].(*int)) / 100000
				returnElement.Lon = float64(*rowContent[1].(*int)) / 100000

				switch rowContent[2].(type) {
				case *string:
					returnElement.ID = *rowContent[2].(*string)
				case *int:
					returnElement.ID = string(*rowContent[2].(*int))
				default:
					return returnElement, KismetDBError("ID data from kismet not proper")
				}

				returnElement.HasData = true

				var extraData []interface{}

				// Check for extra data that will go into the extra data []interface{}
				if numData := len(rowContent) ; numData > 3 {
					returnElement.extraData = true
					extraData = make([]interface{}, numData - 3)

					// This is significantly more complicated than its REST alternative because we get pointer data
					// from the DB call rather than un-referenced data. This means that we have to save it now or
					// loose it forever down the line.
					for n, v := range rowContent[3:] {
						switch v.(type) { // Test type
						case *string: // Match type
							extraData[n] = *(v.(*string))
						case *int:
							extraData[n] = *(v.(*int))
						case *int64:
							extraData[n] = *(v.(*int64))
						case *bool:
							extraData[n] = *(v.(*bool))
						default:
							returnElement.HasData = false
							return returnElement, KismetDBError("Unhandled type in extra data fields")
						}
					}
				} else {
					returnElement.extraData = false // Be explicit
					extraData = nil
				}
				returnElement.data = extraData

				return returnElement, nil
			}
			return returnElement, KismetDBError("No more rows left")
		}, nil
	} else {
		return badFunc, KismetDBError(
			fmt.Sprintf("Failed to run kismet DB query: %v", err))
	}
}

func (client *KismetDBClient) runQuery() error {
	if !client.ready {
		return KismetDBError("DB Client is not read!")
	}
	var query strings.Builder

	columnLen := len(client.columns)
	query.WriteString("select ")
	if columnLen == 0 {
		client.ready = false
		return KismetDBError("No Columns to select from the table")
	} else {
		for i, column := range client.columns {
			if i == columnLen - 1 {
				query.WriteString(fmt.Sprintf("%s ", column))
			} else {
				query.WriteString(fmt.Sprintf("%s, ", column))
			}
		}
	}
	query.WriteString("from " + client.table + ";")
	fmt.Println("Query:", query.String())

	if rows, err := client.db.Query(query.String()) ; err == nil {
		client.rows = rows
		return nil
	} else {
		return KismetDBError(fmt.Sprint("DB Query failed: ", err))
	}
}

// This function returns a fully initialized and ready to run Kismet DB client.
// The client is connected to the database and requires the Finish() call to
// clean up and disconnect from the database when users are finished with it.
func NewDBClient(dbFile, table string, columns []string) (KismetDBClient, error) {
	var (
		db *sql.DB
	)

	if _, err := os.Stat(dbFile) ; os.IsExist(err) {
		return KismetDBClient{}, KismetDBError(fmt.Sprintf("%s does not exist!", dbFile))
	}

	if newDB, err := sql.Open("sqlite3", dbFile) ; err == nil {
		db = newDB
	} else {
		return KismetDBClient{}, KismetDBError(fmt.Sprint("Failed to create DB connection", err))
	}

	return KismetDBClient{
		db,
		nil,
		table,
		columns,
		true,
	}, nil
}
