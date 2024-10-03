package main

import (
	"net/http"
)
import "net/url"

func (s *ProfileParams) Valid(query url.Values) error {
	var err error
	if s.Login, err = validString(
		query,
		"login",
		true,
		nil,
		-9223372036854775808,
		9223372036854775807,
		"",
	); err != nil {
		return err
	}
	return nil
}

func (s *CreateParams) Valid(query url.Values) error {
	var err error
	if s.Login, err = validString(
		query,
		"login",
		true,
		nil,
		10,
		9223372036854775807,
		"",
	); err != nil {
		return err
	}
	if s.Name, err = validString(
		query,
		"full_name",
		false,
		nil,
		-9223372036854775808,
		9223372036854775807,
		"",
	); err != nil {
		return err
	}
	if s.Status, err = validString(
		query,
		"status",
		false,
		[]string{"user", "moderator", "admin"},
		-9223372036854775808,
		9223372036854775807,
		"user",
	); err != nil {
		return err
	}
	if s.Age, err = validInt(
		query,
		"age",
		false,
		nil,
		0,
		128,
	); err != nil {
		return err
	}
	return nil
}

func (s *OtherCreateParams) Valid(query url.Values) error {
	var err error
	if s.Username, err = validString(
		query,
		"username",
		true,
		nil,
		3,
		9223372036854775807,
		"",
	); err != nil {
		return err
	}
	if s.Name, err = validString(
		query,
		"account_name",
		false,
		nil,
		-9223372036854775808,
		9223372036854775807,
		"",
	); err != nil {
		return err
	}
	if s.Class, err = validString(
		query,
		"class",
		false,
		[]string{"warrior", "sorcerer", "rouge"},
		-9223372036854775808,
		9223372036854775807,
		"warrior",
	); err != nil {
		return err
	}
	if s.Level, err = validInt(
		query,
		"level",
		false,
		nil,
		1,
		50,
	); err != nil {
		return err
	}
	return nil
}

func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		requestValues, err := validRequest(w, r, "", false)
		if err != nil {
			MarshalAndWrite(w, err)
			return
		}

		param := ProfileParams{}
		if err := param.Valid(requestValues); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			MarshalAndWrite(w, err)
			return
		}
		response, err := srv.Profile(r.Context(), param)
		if err != nil {
			SetFuncError(w, err)
			return
		}
		MarshalAndWrite(w, &ResponseError{"", response})

	case "/user/create":
		requestValues, err := validRequest(w, r, "POST", true)
		if err != nil {
			MarshalAndWrite(w, err)
			return
		}

		param := CreateParams{}
		if err := param.Valid(requestValues); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			MarshalAndWrite(w, err)
			return
		}
		response, err := srv.Create(r.Context(), param)
		if err != nil {
			SetFuncError(w, err)
			return
		}
		MarshalAndWrite(w, &ResponseError{"", response})

	default:
		w.WriteHeader(http.StatusNotFound)
		MarshalAndWrite(w, &ResponseError{"unknown method", nil})
	}
}

func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/create":
		requestValues, err := validRequest(w, r, "POST", true)
		if err != nil {
			MarshalAndWrite(w, err)
			return
		}

		param := OtherCreateParams{}
		if err := param.Valid(requestValues); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			MarshalAndWrite(w, err)
			return
		}
		response, err := srv.Create(r.Context(), param)
		if err != nil {
			SetFuncError(w, err)
			return
		}
		MarshalAndWrite(w, &ResponseError{"", response})

	default:
		w.WriteHeader(http.StatusNotFound)
		MarshalAndWrite(w, &ResponseError{"unknown method", nil})
	}
}
