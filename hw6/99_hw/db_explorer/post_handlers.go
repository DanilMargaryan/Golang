package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (explorer *DbExplore) HandlePost(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(trimmedPath, "/")

	if len(parts) != 2 {
		MarshalAndWrite(w, http.StatusNotFound, ResultResponse{Error: "unknown url"})
		return
	}

	tableName := parts[0]
	id := parts[1]
	_ = id
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

	var placeholders []string
	var values []interface{}

	for _, field := range explorer.tables[tableName] {
		if field.isAutoIncrement {
			continue
		}

		value, exists := record[field.name]
		if !exists {
			continue
		}

		if !isValidType(value, field.fieldType) && !(value == nil && field.isNull) {
			MarshalAndWrite(w, http.StatusBadRequest,
				ResultResponse{Error: fmt.Sprintf("field %s have invalid type", field.name)})
			return
		}

		placeholders = append(placeholders, fmt.Sprintf("%s = ?", field.name))
		values = append(values, value)
	}

	if len(placeholders) == 0 {
		MarshalAndWrite(w, http.StatusBadRequest,
			ResultResponse{Error: fmt.Sprintf("field %s have invalid type", pk)})
		return
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?",
		tableName,
		strings.Join(placeholders, ", "),
		pk)

	stmt, err := explorer.db.Prepare(query)
	if err != nil {
		MarshalAndWrite(w, http.StatusInternalServerError, ResultResponse{Error: "failed to prepare query"})
		return
	}
	defer stmt.Close()

	values = append(values, id)
	res, err := stmt.Exec(values...)
	if err != nil {
		MarshalAndWrite(w, http.StatusInternalServerError, ResultResponse{Error: "failed to execute query"})
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		MarshalAndWrite(w, http.StatusInternalServerError, ResultResponse{Error: "failed to retrieve inserted ID"})
		return
	}

	MarshalAndWrite(w, http.StatusOK, ResultResponse{Response: map[string]interface{}{"updated": rowsAffected}})
}
