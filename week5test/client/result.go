package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)


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


func (srv *MyApi) ProfileWrapped(w http.ResponseWriter, r *http.Request) {
	var q url.Values
	err := errors.New("")
	// заполнение структуры params
	params := ProfileParams{}
	if r.Method == "GET" {
		q = r.URL.Query()
	} else {
		reqBody, _ := ioutil.ReadAll(r.Body)
		q, _ = url.ParseQuery(string(reqBody))
	}
	//Login
	params.Login = q.Get("login") 
	if params.Login == "" {
		sendError(w, "login must me not empty", http.StatusBadRequest)
		return	
	}


	user, err := srv.Profile(context.TODO(), params)
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
	var q url.Values
	err := errors.New("")
	// заполнение структуры params
	params := CreateParams{}
	if r.Method == "GET" {
		q = r.URL.Query()
	} else {
		reqBody, _ := ioutil.ReadAll(r.Body)
		q, _ = url.ParseQuery(string(reqBody))
	}
	//Login
	params.Login = q.Get("login") 
	if params.Login == "" {
		sendError(w, "login must me not empty", http.StatusBadRequest)
		return	
	}
	if len(params.Login) < 10 {
		sendError(w, "login len must be >= 10", http.StatusBadRequest)
		return	
	}//Name
	params.Name = q.Get("full_name") 
	//Status
	params.Status = q.Get("status") 
	if params.Status == "" {
		params.Status = "user"
	}
	enumStatusValid := false
	enumStatus := []string{"user", "moderator", "admin"}
	for _, valid := range enumStatus {
		if valid == params.Status {
			enumStatusValid = true
			break
		}
	}

	if !enumStatusValid {
		sendError(w, "status must be one of " + strings.Join(enumStatus, ", "), http.StatusBadRequest)
		return
	}
	//Age
	params.Age, err = strconv.Atoi(q.Get("age"))
	if err != nil {
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

func (srv *OtherApi) CreateWrapped(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Auth") != "100500" {
		sendError(w, "unauthorized", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		sendError(w, "bad method", http.StatusNotAcceptable)
		return
	}
	var q url.Values
	err := errors.New("")
	// заполнение структуры params
	params := OtherCreateParams{}
	if r.Method == "GET" {
		q = r.URL.Query()
	} else {
		reqBody, _ := ioutil.ReadAll(r.Body)
		q, _ = url.ParseQuery(string(reqBody))
	}
	//Username
	params.Username = q.Get("username") 
	if params.Username == "" {
		sendError(w, "username must me not empty", http.StatusBadRequest)
		return	
	}
	if len(params.Username) < 3 {
		sendError(w, "username len must be >= 3", http.StatusBadRequest)
		return	
	}//Name
	params.Name = q.Get("account_name") 
	//Class
	params.Class = q.Get("class") 
	if params.Class == "" {
		params.Class = "warrior"
	}
	enumClassValid := false
	enumClass := []string{"warrior", "sorcerer", "rouge"}
	for _, valid := range enumClass {
		if valid == params.Class {
			enumClassValid = true
			break
		}
	}

	if !enumClassValid {
		sendError(w, "class must be one of " + strings.Join(enumClass, ", "), http.StatusBadRequest)
		return
	}
	//Level
	params.Level, err = strconv.Atoi(q.Get("level"))
	if err != nil {
		sendError(w, "age must be int", http.StatusBadRequest)
		return	
	}
	if params.Level < 1 {
		sendError(w, "level must be >= 1", http.StatusBadRequest)
		return	
	}
	if params.Level > 50 {
		sendError(w, "level must be <= 50", http.StatusBadRequest)
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

func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	
	case "/user/profile":
		srv.ProfileWrapped(w, r)
	
	case "/user/create":
		srv.CreateWrapped(w, r)
	default:
		sendError(w, "unknown method", http.StatusNotFound)
	}
}

func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	
	case "/user/create":
		srv.CreateWrapped(w, r)
	default:
		sendError(w, "unknown method", http.StatusNotFound)
	}
}
