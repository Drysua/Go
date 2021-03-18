package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

func SetVal(list []string) string {
	if len(list) != 0 {
		return list[0]
	}
	return ""
}

func MustAtoi(s string) int {
	res := 0
	err := errors.New("")
	if s != "" {
		res, err = strconv.Atoi(s)
		if err != nil {
			return -42
		}
	}
	return res
}

func GetParams(r *http.Request, w http.ResponseWriter) CreateParams {
	var params CreateParams
	if r.Method == "GET" {
		q := r.URL.Query()
		params = CreateParams{
			Login:  SetVal(q["login"]),
			Name:   SetVal(q["full_name"]),
			Status: SetVal(q["status"]),
			Age:    MustAtoi(SetVal(q["age"])),
		}
	} else {
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		q, err := url.ParseQuery(string(reqBody))
		if err != nil {
			log.Fatal(err)
		}
		params = CreateParams{
			Login:  q.Get("login"),
			Name:   q.Get("full_name"),
			Status: q.Get("status"),
			Age:    MustAtoi(q.Get("age")),
		}
	}
	return params
}

func sendError(w http.ResponseWriter, error string, code int) {
	js, err := json.Marshal(CR{"error": error})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintln(w, string(js))
}

func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case ApiUserProfile:
		srv.ProfileWrapped(w, r)
	case ApiUserCreate:
		srv.CreateWrapped(w, r)
	default:
		sendError(w, "unknown method", http.StatusNotFound)
	}
}

func (srv *MyApi) ProfileWrapped(w http.ResponseWriter, r *http.Request) {
	// заполнение структуры params
	params := GetParams(r, w)
	// валидирование параметров
	// логин не должен быть пустым
	if params.Login == "" {
		sendError(w, "login must me not empty", http.StatusBadRequest)
		return
	}
	if params.Age == -42 {
		sendError(w, "age must be int", http.StatusBadRequest)
		return
	}
	if params.Age < 0 {
		sendError(w, "age must be >= 0", http.StatusBadRequest)
		return
	}
	if params.Age > 128 {
		sendError(w, "age must be <= 128", http.StatusBadRequest)
		return
	}
	if !(params.Status == "user" || params.Status == "moderator" || params.Status == "admin") {
		sendError(w, "status must be one of [user, moderator, admin]", http.StatusBadRequest)
		return
	}
	user, err := srv.Profile(context.TODO(), ProfileParams{params.Login})
	if err != nil {
		switch err := err.(type) {
		case ApiError:
			sendError(w, err.Error(), err.HTTPStatus)
		default:
			sendError(w, err.Error(), http.StatusBadRequest)
		}
		return
	}
	result := CR{
		"error":    "",
		"response": user,
	}

	b, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, string(b))
}

func (srv *MyApi) CreateWrapped(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Auth") != "100500" {
		sendError(w, "unauthorized", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		sendError(w, "bad method", http.StatusNotAcceptable)
		return
	}
	params := GetParams(r, w)
	if params.Login == "" {
		sendError(w, "login must me not empty", http.StatusBadRequest)
		return
	}
	if params.Age == -42 {
		sendError(w, "age must be int", http.StatusBadRequest)
		return
	}
	if params.Age < 0 {
		sendError(w, "age must be >= 0", http.StatusBadRequest)
		return
	}
	if params.Age > 128 {
		sendError(w, "age must be <= 128", http.StatusBadRequest)
		return
	}
	if params.Status == "" {
		params.Status = "user"
	}
	if !(params.Status == "user" || params.Status == "moderator" || params.Status == "admin") {
		sendError(w, "status must be one of [user, moderator, admin]", http.StatusBadRequest)
		return
	}
	if len(params.Login) < 10 {
		sendError(w, "login len must be >= 10", http.StatusBadRequest)
		return
	}
	user, err := srv.Create(context.TODO(), params)
	if err != nil {
		switch err := err.(type) {
		case ApiError:
			sendError(w, err.Error(), err.HTTPStatus)
		default:
			sendError(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	// прочие обработки

	result := CR{
		"error":    "",
		"response": user,
	}

	b, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, string(b))
}
