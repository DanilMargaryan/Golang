package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type ResponseError struct {
	ErrorText string      `json:"error"`
	Response  interface{} `json:"response,omitempty"`
}

func (e *ResponseError) Error() string {
	return e.ErrorText
}

func validString(
	values url.Values,
	key string,
	isRequire bool,
	enums []string,
	minValue int,
	maxValue int,
	defaultValue string,
) (string, error) {
	value := values.Get(key)
	if len(value) == 0 {
		value = defaultValue
	}

	if isRequire && len(value) == 0 {
		return "", &ResponseError{fmt.Sprintf("%s must be not empty", key), nil}
	}

	valueRunes := []rune(value)
	if maxValue < len(valueRunes) {
		return value, &ResponseError{fmt.Sprintf("%s len must be <= %d", key, maxValue), nil}
	}

	if minValue > len(valueRunes) {
		return value, &ResponseError{fmt.Sprintf("%s len must be >= %d", key, minValue), nil}
	}

	if len(enums) == 0 {
		return value, nil
	}

	for _, enum := range enums {
		if enum == value {
			return value, nil
		}
	}
	return value, &ResponseError{fmt.Sprintf("%s must be one of %v", key, enums), nil}
}

func validInt(
	values url.Values,
	key string,
	isRequire bool,
	enums []int,
	minValue int,
	maxValue int,
) (int, error) {
	value := values.Get(key)

	if len(value) == 0 && isRequire {
		return 0, &ResponseError{fmt.Sprintf("%s must me not empty", key), nil}
	}

	val, err := strconv.Atoi(value)
	if err != nil {
		return 0, &ResponseError{fmt.Sprintf("%s must be int", key), nil}
	}

	if maxValue < val {
		return val, &ResponseError{fmt.Sprintf("%s must be <= %d", key, maxValue), nil}
	}

	if minValue > val {
		return val, &ResponseError{fmt.Sprintf("%s must be >= %d", key, minValue), nil}
	}

	if len(enums) == 0 {
		return val, nil
	}

	for _, enum := range enums {
		if enum == val {
			return val, nil
		}
	}

	return 0, &ResponseError{fmt.Sprintf("%s must me not empty", key), nil}
}

func MarshalAndWrite(w http.ResponseWriter, v interface{}) {
	response, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	w.Write(response)
}

func SetFuncError(w http.ResponseWriter, err error) {
	var mErr ApiError
	switch {
	case errors.As(err, &mErr):
		w.WriteHeader(mErr.HTTPStatus)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	MarshalAndWrite(w, &ResponseError{err.Error(), nil})
}

func validRequest(w http.ResponseWriter, r *http.Request, expectedMethod string, auth bool,
) (url.Values, error) {
	if expectedMethod != "" && r.Method != expectedMethod {
		w.WriteHeader(http.StatusNotAcceptable)
		return nil, &ResponseError{"bad method", nil}
	}
	if r.Header.Get("X-Auth") != "100500" && auth {
		w.WriteHeader(http.StatusForbidden)
		return nil, &ResponseError{"unauthorized", nil}
	}
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return nil, &ResponseError{"value error", nil}
		}
		return r.Form, nil
	}
	return r.URL.Query(), nil
}
