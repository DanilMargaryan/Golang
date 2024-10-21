package main

import (
	"database/sql"
	"fmt"
	"net/http"
)

type DbExplore struct {
	db     *sql.DB
	tables map[string][]Field
}

type Field struct {
	name            string
	fieldType       string
	isNull          bool
	isAutoIncrement bool
	isPrimaryKey    bool
}

func (explorer *DbExplore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		explorer.HandleGet(w, r)
	case http.MethodPost:
		explorer.HandlePost(w, r)
	case http.MethodPut:
		explorer.HandlePut(w, r)
	case http.MethodDelete:
		explorer.HandleDelete(w, r)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func NewDbExplorer(db *sql.DB) (*DbExplore, error) {
	allTables := make(map[string][]Field)

	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, fmt.Errorf("show tables: %s", err)
	}

	var tableName string
	var tableNames []string
	for rows.Next() {
		if err = rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("scan table name: %s", err)
		}
		tableNames = append(tableNames, tableName)
	}
	rows.Close()

	for _, tableName = range tableNames {
		query := fmt.Sprintf("SHOW FULL COLUMNS FROM %s", tableName)
		columns, err := db.Query(query)
		if err != nil {
			return nil, fmt.Errorf("show columns: %s", err)
		}

		var columnName, colType, isNull, key string
		var isNullBool, isAutoIncrement, isPrimaryKey bool
		var collation, defaultValue, extra, privileges, comment sql.NullString
		var columnList []Field

		for columns.Next() {
			if err := columns.Scan(&columnName, &colType, &collation, &isNull, &key, &defaultValue, &extra, &privileges, &comment); err != nil {
				return nil, fmt.Errorf("scan column for table %s: %s", tableName, err)
			}

			if isNull == "YES" {
				isNullBool = true
			} else {
				isNullBool = false
			}

			if extra.Valid && extra.String == "auto_increment" {
				isAutoIncrement = true
			} else {
				isAutoIncrement = false
			}

			if key == "PRI" {
				isPrimaryKey = true
			} else {
				isPrimaryKey = false
			}

			columnList = append(columnList, Field{columnName, colType, isNullBool, isAutoIncrement, isPrimaryKey})
		}
		columns.Close()

		allTables[tableName] = columnList
	}

	return &DbExplore{db: db, tables: allTables}, nil
}
