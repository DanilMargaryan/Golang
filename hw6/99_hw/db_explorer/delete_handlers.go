package main

import (
	"fmt"
	"net/http"
	"strings"
)

func (explorer *DbExplore) HandleDelete(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(trimmedPath, "/")

	if len(parts) != 2 {
		MarshalAndWrite(w, http.StatusNotFound, ResultResponse{Error: "unknown url"})
		return
	}

	tableName := parts[0]
	id := parts[1]
	_ = id
	if _, exists := explorer.tables[tableName]; !exists {
		MarshalAndWrite(w, http.StatusNotFound, ResultResponse{Error: "unknown tableName"})
		return
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableName)

	stmt, err := explorer.db.Prepare(query)
	if err != nil {
		MarshalAndWrite(w, http.StatusInternalServerError, ResultResponse{Error: "failed to prepare query"})
		return
	}
	defer stmt.Close()

	res, err := stmt.Exec(id)
	if err != nil {
		MarshalAndWrite(w, http.StatusInternalServerError, ResultResponse{Error: "failed to execute query"})
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		MarshalAndWrite(w, http.StatusInternalServerError, ResultResponse{Error: "failed to retrieve inserted ID"})
		return
	}

	MarshalAndWrite(w, http.StatusOK, ResultResponse{Response: map[string]interface{}{"deleted": rowsAffected}})
}
