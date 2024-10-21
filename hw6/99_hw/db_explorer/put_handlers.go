package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (explorer *DbExplore) HandlePut(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(trimmedPath, "/")

	if len(parts) != 1 {
		MarshalAndWrite(w, http.StatusNotFound, ResultResponse{Error: "unknown url"})
		return
	}
	tableName := parts[0]
	pk, _ := explorer.getPrimaryKey(tableName)

	if _, exists := explorer.tables[tableName]; !exists {
		MarshalAndWrite(w, http.StatusNotFound, ResultResponse{Error: "unknown tableName"})
		return
	}

	defer r.Body.Close()
	body, _ := io.ReadAll(r.Body)

	record := make(map[string]interface{})
	if err := json.Unmarshal(body, &record); err != nil {
		MarshalAndWrite(w, http.StatusBadRequest, ResultResponse{Error: "invalid JSON format"})
		return
	}

	var columns []string
	var placeholders []string
	var values []interface{}

	for _, field := range explorer.tables[tableName] {
		if field.isAutoIncrement {
			continue
		}

		value, exists := record[field.name]
		if !exists && !field.isNull {
			switch field.fieldType {
			case "int", "int(11)":
				value = 0
			case "varchar", "text", "varchar(255)": // строки
				value = ""
			}
		}

		if !isValidType(value, field.fieldType) && !(value == nil && field.isNull) {
			MarshalAndWrite(w, http.StatusBadRequest,
				ResultResponse{Error: fmt.Sprintf("Invalid type for field %s: expected %s", field.name, field.fieldType)})
			return
		}

		columns = append(columns, field.name)
		placeholders = append(placeholders, "?")
		values = append(values, value)
	}

	query := fmt.Sprintf("INSERt INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ","),
		strings.Join(placeholders, ","))

	stmt, err := explorer.db.Prepare(query)
	if err != nil {
		fmt.Println(err)
		MarshalAndWrite(w, http.StatusInternalServerError, ResultResponse{Error: "failed to prepare query"})
		return
	}
	defer stmt.Close()

	id, err := stmt.Exec(values...)
	if err != nil {
		MarshalAndWrite(w, http.StatusInternalServerError, ResultResponse{Error: "failed to execute query"})
		return
	}

	insertedID, err := id.LastInsertId()
	if err != nil {
		MarshalAndWrite(w, http.StatusInternalServerError, ResultResponse{Error: "failed to retrieve inserted ID"})
		return
	}

	MarshalAndWrite(w, http.StatusOK, ResultResponse{Response: map[string]interface{}{fmt.Sprintf("%s", pk): insertedID}})
}
