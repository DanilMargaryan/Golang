package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Person struct {
	Name      string `xml:"-"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Id        int    `xml:"id"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

const (
	fieldName = "name"
	fieldId   = "id"
	fieldAge  = "age"
)

const accessToken = "clown_token"

type Root struct {
	Persons []Person `xml:"row"`
}

var root Root

func Parse() {
	byteValue, err := os.ReadFile("dataset.xml")
	if err != nil {
		fmt.Printf("Error reading file: %s", err)
	}

	err = xml.Unmarshal(byteValue, &root)
	if err != nil {
		fmt.Printf("Error unmarshaling XML: %s", err)
	}

	for i, person := range root.Persons {
		root.Persons[i].Name = fmt.Sprintf("%s %s", person.FirstName, person.LastName)
	}
}

func SortBy(persons *[]Person, field string, by int) error {
	if len(field) == 0 {
		field = "name"
	}
	switch field {
	case fieldName:
		if by == OrderByAsc {
			sort.Slice(*persons, func(i, j int) bool { return (*persons)[i].Name < (*persons)[j].Name })
		} else {
			sort.Slice(*persons, func(i, j int) bool { return (*persons)[i].Name > (*persons)[j].Name })
		}
	case fieldId:
		if by == OrderByAsc {
			sort.Slice(*persons, func(i, j int) bool { return (*persons)[i].Id < (*persons)[j].Id })
		} else {
			sort.Slice(*persons, func(i, j int) bool { return (*persons)[i].Id > (*persons)[j].Id })
		}
	case fieldAge:
		if by == OrderByAsc {
			sort.Slice(*persons, func(i, j int) bool { return (*persons)[i].Age < (*persons)[j].Age })
		} else {
			sort.Slice(*persons, func(i, j int) bool { return (*persons)[i].Age > (*persons)[j].Age })
		}
	default:
		return errors.New(ErrorBadOrderField)
	}
	return nil
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	var persons []Person

	if accessToken != r.Header.Get("AccessToken") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("query")
	orderField := r.URL.Query().Get("order_field")
	orderBy := r.URL.Query().Get("order_by")
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")

	for _, person := range root.Persons {
		if strings.Contains(person.Name, query) || strings.Contains(person.About, query) {
			persons = append(persons, person)
		}
	}

	order, err := strconv.Atoi(orderBy)
	if err != nil || order < -1 || order > 1 {
		http.Error(w, "order incorrect", http.StatusBadRequest)
	}

	err = SortBy(&persons, orderField, order)
	if err != nil {
		response := &SearchErrorResponse{Error: err.Error()}
		w.WriteHeader(http.StatusBadRequest)
		errorText, _ := json.Marshal(response)
		_, _ = w.Write(errorText)
		return
	}

	offsetInt, err := strconv.Atoi(offset)
	if err != nil || offsetInt < 0 {
		offsetInt = 0
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt <= 0 {
		limitInt = len(persons)
	}

	result := persons[offsetInt:limitInt]

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		http.Error(w, "Internal Server Error: unable to encode JSON", http.StatusInternalServerError)
		return
	}
}

func main() {
	Parse()
	http.HandleFunc("/", SearchServer)
	_ = http.ListenAndServe(":8080", nil)
}
