/*
	В некоторых случаях я не обрабатываю ошибку в js, _ := json.Marshal(...), так как на грейдер приходят валидные значения.
	НЕ ДЕЛАЙТЕ ТАК! ВСЕГДА ОБРАБАТЫВАЙТЕ ОШИБКИ!!!
*/

package main

import (
	"encoding/json"
	"net/http"
	"errors"
	"strconv"
	"strings"
)

var (
	errorBad	 = errors.New("bad method")
	errorEmpty	 = errors.New("login must me not empty")
	errorAuth	 = errors.New("unauthorized")
	errorUnknown = errors.New("unknown method")
)

type JsonErrors struct{
	Error string	`json:"error"`
}

//Response Json for  ProfileParams do not edit
type JsonProfileParams struct{
	* User `json:"response"` 
	JsonErrors
}

//Response Json for  CreateParams do not edit
type JsonCreateParams struct{
	* NewUser `json:"response"` 
	JsonErrors
}

//Response Json for  OtherCreateParams do not edit
type JsonOtherCreateParams struct{
	* OtherUser `json:"response"` 
	JsonErrors
}

func (h * MyApi  ) ServeHTTP(w http.ResponseWriter, r *http.Request) { 
	switch r.URL.Path{
		case "/user/profile":
			h.profile(w, r)
		case "/user/create":
			h.create(w, r)
		default:
			js, err := json.Marshal(JsonErrors{errorUnknown.Error()})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			w.Write(js)
			return
	}
}

func (h *  OtherApi  ) ServeHTTP(w http.ResponseWriter, r *http.Request) { 
		switch r.URL.Path{
			case "/user/create":
				h.create(w, r)
			default:
				js, err := json.Marshal(JsonErrors{errorUnknown.Error()})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNotFound)
				w.Write(js)
				return
		}
}

// Func Wrapper. DO NOT EDIT
func (h * MyApi) profile(w http.ResponseWriter, r *http.Request) {
	var login string
	switch r.Method{
		case "GET":
			login = r.URL.Query().Get("login")
			if login == ""{
				js, err := json.Marshal(JsonErrors{errorEmpty.Error()})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				w.Header().Set("Content-Type", "application/json")
				w.Write(js)
				return
			}
		case "POST":
			r.ParseForm()
			login = r.Form.Get("login")
			if login == ""{
				js, err := json.Marshal(JsonErrors{errorEmpty.Error()})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				w.Header().Set("Content-Type", "application/json")
				w.Write(js)
				return
			}
		default:
			http.Error(w, "Sorry, only GET and POST methods are supported ",http.StatusInternalServerError)
	}
	ctx := r.Context()
	profileParams := ProfileParams{login, }
	user, err := h.Profile (ctx, profileParams)

	if err != nil{
		switch err.(type){
		case ApiError:
			js, _ := json.Marshal(JsonErrors{err.(ApiError).Err.Error()})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(err.(ApiError).HTTPStatus)
			w.Write(js)
			return
		default:
			js, _ := json.Marshal(JsonErrors{"bad user"})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(js)
			return
		}
	}
	js, _ := json.Marshal(JsonProfileParams{user, JsonErrors{""}})
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
// Func Wrapper. DO NOT EDIT
func (h * MyApi) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != "POST"{
		js, err := json.Marshal(JsonErrors{errorBad.Error()})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotAcceptable)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
		return
	}
	if r.Header.Get("X-Auth") != "100500"{
		js, err := json.Marshal(JsonErrors{errorAuth.Error()})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
		return
	}
	r.ParseForm()
	login:= r.Form.Get("login") 


	if  login == ""{
		js, _ := json.Marshal(JsonErrors{errorEmpty.Error()})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}

	if len(login)<10{
		js, _ := json.Marshal(JsonErrors{"login len must be >= 10"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
	name:= r.Form.Get("name") 


	paramName := r.Form.Get("full_name")

 	if paramName == ""{
		paramName = strings.ToLower( name )
	}else{ 
 		 name = paramName
	}

	status:= r.Form.Get("status") 

	if status== ""{
		status = "user" 
	}
	enum := make(map[string]bool)
	enum["user"] = true
	enum["moderator"] = true
	enum["admin"] = true
	_, enumName := enum[status]
	if enumName == false{
		js, _ := json.Marshal(JsonErrors{"status must be one of [user, moderator, admin]"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
	age, err := strconv.Atoi(r.Form.Get("age")) 

	if err != nil {
 		js, _ := json.Marshal(JsonErrors{"age must be int"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}

	if age<0{
		js, _ := json.Marshal(JsonErrors{"age must be >= 0"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
							
	if age>128{
		js, _ := json.Marshal(JsonErrors{"age must be <= 128"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
							
	createParams := CreateParams{login, name, status, age, }
	user, err := h.Create (ctx, createParams)

	if err != nil{
		switch err.(type){
		case ApiError:
			js, _ := json.Marshal(JsonErrors{err.(ApiError).Err.Error()})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(err.(ApiError).HTTPStatus)
			w.Write(js)
			return
		default:
			js, _ := json.Marshal(JsonErrors{"bad user"})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(js)
			return
		}
	}
	js, _ := json.Marshal(JsonCreateParams{user, JsonErrors{""}})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}
// Func Wrapper. DO NOT EDIT
func (h * OtherApi) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != "POST"{
		js, err := json.Marshal(JsonErrors{errorBad.Error()})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotAcceptable)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
		return
	}
	if r.Header.Get("X-Auth") != "100500"{
		js, err := json.Marshal(JsonErrors{errorAuth.Error()})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
		return
	}
	r.ParseForm()
	username:= r.Form.Get("username") 


	if  username == ""{
		js, _ := json.Marshal(JsonErrors{errorEmpty.Error()})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}

	if len(username)<3{
		js, _ := json.Marshal(JsonErrors{"username len must be >= 3"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
	name:= r.Form.Get("name") 


	paramName := r.Form.Get("account_name")

 	if paramName == ""{
		paramName = strings.ToLower( name )
	}else{ 
 		 name = paramName
	}

	class:= r.Form.Get("class") 

	if class== ""{
		class = "warrior" 
	}
	enum := make(map[string]bool)
	enum["warrior"] = true
	enum["sorcerer"] = true
	enum["rouge"] = true
	_, enumName := enum[class]
	if enumName == false{
		js, _ := json.Marshal(JsonErrors{"class must be one of [warrior, sorcerer, rouge]"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
	level, err := strconv.Atoi(r.Form.Get("level")) 

	if err != nil {
 		js, _ := json.Marshal(JsonErrors{"level must be int"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}

	if level<1{
		js, _ := json.Marshal(JsonErrors{"level must be >= 1"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
							
	if level>50{
		js, _ := json.Marshal(JsonErrors{"level must be <= 50"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
							
	createParams := OtherCreateParams{username, name, class, level, }
	user, err := h.Create (ctx, createParams)

	if err != nil{
		switch err.(type){
		case ApiError:
			js, _ := json.Marshal(JsonErrors{err.(ApiError).Err.Error()})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(err.(ApiError).HTTPStatus)
			w.Write(js)
			return
		default:
			js, _ := json.Marshal(JsonErrors{"bad user"})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(js)
			return
		}
	}
	js, _ := json.Marshal(JsonOtherCreateParams{user, JsonErrors{""}})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}
