package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (explorer *DbExplore) HandleGet(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.Trim(r.URL.Path, "/")
	if trimmedPath == "" {
		explorer.getTables(w, r)
		return
	}

	parts := strings.Split(trimmedPath, "/")

	switch len(parts) {
	case 1:
		explorer.getRecords(w, r, parts[0])
	case 2:
		explorer.getRecordById(w, r, parts[0], parts[1])
	default:
		http.Error(w, "404 Not Found", http.StatusNotFound)
	}
}

func (explorer *DbExplore) getTables(w http.ResponseWriter, r *http.Request) {
	tableNames := make([]string, 0, len(explorer.tables))
	for table := range explorer.tables {
		tableNames = append(tableNames, table)
	}

	MarshalAndWrite(w, http.StatusOK, ResultResponse{Response: map[string][]string{"tables": tableNames}})
}

func (explorer *DbExplore) getRecords(w http.ResponseWriter, r *http.Request, table string) {
	limit := 5
	offset := 0

	if _, exists := explorer.tables[table]; !exists {
		MarshalAndWrite(w, http.StatusNotFound, ResultResponse{Error: "unknown table"})
		return
	}

	queryParams := r.URL.Query()

	if l := queryParams.Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil {
			limit = parsedLimit
		}
	}

	if o := queryParams.Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil {
			offset = parsedOffset
		}
	}

	query := fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", table)
	rows, err := explorer.db.Query(query, limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	results, err := getRows(rows)
	if err != nil {
		fmt.Errorf("getRows: %s", err.Error())
	}
	MarshalAndWrite(w, http.StatusOK, ResultResponse{Response: map[string]interface{}{"records": results}})
}

func (explorer *DbExplore) getRecordById(w http.ResponseWriter, r *http.Request, table string, idStr string) {
	if _, exists := explorer.tables[table]; !exists {
		MarshalAndWrite(w, http.StatusNotFound, ResultResponse{Error: "unknown table"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		fmt.Println(err)
	}

	pk, _ := explorer.getPrimaryKey(table)

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", table, pk)
	rows, err := explorer.db.Query(query, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	results, err := getRows(rows)
	if err != nil {
		fmt.Errorf("getRows: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(results) == 0 {
		MarshalAndWrite(w, http.StatusNotFound, ResultResponse{Error: "record not found"})
		return
	}
	result := results[0]
	MarshalAndWrite(w, http.StatusOK, ResultResponse{Response: map[string]interface{}{"record": result}})
}
