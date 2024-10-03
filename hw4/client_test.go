package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type TestCase struct {
	searchClient   SearchClient
	searchRequest  SearchRequest
	searchResponse SearchResponse
	expectedError  string
}

func TestSearchClient_LimitErr(t *testing.T) {
	cases := []TestCase{
		TestCase{
			searchClient: SearchClient{
				AccessToken: accessToken,
				URL:         "http://127.0.0.1:8080/",
			},
			searchRequest: SearchRequest{
				Limit:      -5,
				Offset:     0,
				Query:      "",
				OrderField: "name",
				OrderBy:    OrderByAsc,
			},
			expectedError: "limit must be > 0",
		},

		TestCase{
			searchClient: SearchClient{
				AccessToken: accessToken,
				URL:         "http://127.0.0.1:8080/",
			},
			searchRequest: SearchRequest{
				Limit:      100,
				Offset:     0,
				Query:      "",
				OrderField: "name",
				OrderBy:    OrderByAsc,
			},
			expectedError: "",
		},

		TestCase{
			searchClient: SearchClient{
				AccessToken: accessToken,
				URL:         "http://127.0.0.1:8080/",
			},
			searchRequest: SearchRequest{
				Limit:      1,
				Offset:     -10,
				Query:      "",
				OrderField: "name",
				OrderBy:    OrderByAsc,
			},
			expectedError: "offset must be > 0",
		},

		TestCase{
			searchClient: SearchClient{
				AccessToken: accessToken,
				URL:         "http://127.0.0.1:8080/",
			},
			searchRequest: SearchRequest{
				Limit:      5,
				Offset:     2,
				Query:      "",
				OrderField: "name",
				OrderBy:    OrderByAsc,
			},
			expectedError: "",
		},
	}

	for caseNum, item := range cases {
		_, err := item.searchClient.FindUsers(item.searchRequest)
		if item.expectedError == "" && err != nil {
			t.Errorf("case %d: unexpected error: %v", caseNum, err)
		} else if item.expectedError != "" && (err == nil || err.Error() != item.expectedError) {
			t.Errorf("case %d: expected error %q, but got %v", caseNum, item.expectedError, err)
		} else {
			fmt.Printf("Test case %d passed.\n", caseNum)
		}
	}
}

func TestSearchClient_Unauthorized(t *testing.T) {
	testCase := TestCase{
		searchClient: SearchClient{
			AccessToken: "badToken",
			URL:         "http://127.0.0.1:8080/",
		},
		expectedError: "Bad AccessToken",
	}
	_, err := testCase.searchClient.FindUsers(testCase.searchRequest)
	if err == nil || err.Error() != testCase.expectedError {
		t.Errorf("expected error %q, but got %v", testCase.expectedError, err)
	}
}

func TestSearchClient_InternalServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(StatusInternalServerErrorSearchClient))

	testCase := TestCase{
		searchClient: SearchClient{
			AccessToken: accessToken,
			URL:         ts.URL,
		},
		expectedError: "SearchServer fatal error",
	}

	_, err := testCase.searchClient.FindUsers(testCase.searchRequest)

	if err == nil || err.Error() != testCase.expectedError {
		t.Errorf("expected error %q, but got %v", testCase.expectedError, err)
	}
}

func StatusInternalServerErrorSearchClient(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func TestSearchClient_BadRequest1(t *testing.T) {
	testCase := TestCase{
		searchClient: SearchClient{
			AccessToken: accessToken,
			URL:         "http://127.0.0.1:8080/",
		},
		searchRequest: SearchRequest{
			Limit:      10,
			Offset:     0,
			Query:      "",
			OrderField: "FAIL",
			OrderBy:    OrderByAsc,
		},
	}
	_, err := testCase.searchClient.FindUsers(testCase.searchRequest)
	checkoutErr := fmt.Sprintf("OrderFeld %s invalid", testCase.searchRequest.OrderField)
	if err == nil || err.Error() != checkoutErr {
		t.Errorf("expected error %q, but got %v", testCase.expectedError, err)
	}

	testCase2 := TestCase{
		searchRequest: SearchRequest{
			Limit:      10,
			Offset:     0,
			Query:      "",
			OrderField: "name",
			OrderBy:    100,
		},
		expectedError: "cant unpack error json:",
	}
	_, err = testCase.searchClient.FindUsers(testCase2.searchRequest)
	if err == nil || !strings.Contains(err.Error(), testCase2.expectedError) {
		t.Errorf("expected error %q, but got %v", testCase2.expectedError, err)
	}
}

func TestSearchClient_jsonUnmarshal(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(unpackJson))

	testCase := TestCase{
		searchClient: SearchClient{
			AccessToken: accessToken,
			URL:         ts.URL,
		},
		expectedError: "cant unpack result json:",
	}

	_, err := testCase.searchClient.FindUsers(testCase.searchRequest)

	if err == nil || !strings.Contains(err.Error(), testCase.expectedError) {
		t.Errorf("expected error %q, but got %v", testCase.expectedError, err)
	}
}

func unpackJson(w http.ResponseWriter, r *http.Request) {
	badJson := ""

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(badJson)
}

func TestSearchClient_badUrl(t *testing.T) {
	testCase := TestCase{
		searchClient: SearchClient{
			AccessToken: accessToken,
			URL:         "FAIL",
		},
		expectedError: "unknown error",
	}

	_, err := testCase.searchClient.FindUsers(testCase.searchRequest)
	if err == nil || !strings.Contains(err.Error(), testCase.expectedError) {
		t.Errorf("expected error %q, but got %v", testCase.expectedError, err)
	}
}

func TestSearchClient_timeOut(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(sleepFunc))

	testCase := TestCase{
		searchClient: SearchClient{
			AccessToken: accessToken,
			URL:         ts.URL,
		},
		expectedError: "timeout for ",
	}
	_, err := testCase.searchClient.FindUsers(testCase.searchRequest)
	if err == nil || !strings.Contains(err.Error(), testCase.expectedError) {
		t.Errorf("expected error %q, but got %v", testCase.expectedError, err)
	}
}

func sleepFunc(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Second * 2)
}
