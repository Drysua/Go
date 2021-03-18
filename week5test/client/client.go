package main

// это программа для которой ваш кодогенератор будет писать код
// запускать через go test -v, как обычно

// этот код закомментирован чтобы он не светился в тестовом покрытии

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	client = &http.Client{Timeout: time.Second}
)

const (
	ApiUserCreate  = "/user/create"
	ApiUserProfile = "/user/profile"
)

type ApiError struct {
	HTTPStatus int
	Err        error
}

func (ae ApiError) Error() string {
	return ae.Err.Error()
}

const (
	statusUser      = 0
	statusModerator = 10
	statusAdmin     = 20
)

type MyApi struct {
	statuses map[string]int
	users    map[string]*User
	nextID   uint64
	mu       *sync.RWMutex
}
type CreateParams struct {
	Login  string `apivalidator:"required,min=10"`
	Name   string `apivalidator:"paramname=full_name"`
	Status string `apivalidator:"enum=user|moderator|admin,default=user"`
	Age    int    `apivalidator:"min=0,max=128"`
}

type ProfileParams struct {
	Login string `apivalidator:"required"`
}

type User struct {
	ID       uint64 `json:"id"`
	Login    string `json:"login"`
	FullName string `json:"full_name"`
	Status   int    `json:"status"`
}

type NewUser struct {
	ID uint64 `json:"id"`
}

func NewMyApi() *MyApi {
	return &MyApi{
		statuses: map[string]int{
			"user":      0,
			"moderator": 10,
			"admin":     20,
		},
		users: map[string]*User{
			"rvasily": &User{
				ID:       42,
				Login:    "rvasily",
				FullName: "Vasily Romanov",
				Status:   statusAdmin,
			},
		},
		nextID: 43,
		mu:     &sync.RWMutex{},
	}
}

type Case struct {
	Method string // GET по-умолчанию в http.NewRequest если передали пустую строку
	Path   string
	Query  string
	Auth   bool
	Status int
	Result interface{}
}

// CaseResponse
type CR map[string]interface{}

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

// apigen:api {"url": "/user/profile", "auth": false}
func (srv *MyApi) Profile(ctx context.Context, in ProfileParams) (*User, error) {

	if in.Login == "bad_user" {
		return nil, fmt.Errorf("bad user")
	}

	srv.mu.RLock()
	user, exist := srv.users[in.Login]
	srv.mu.RUnlock()
	if !exist {
		return nil, ApiError{http.StatusNotFound, fmt.Errorf("user not exist")}
	}

	return user, nil
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
	// fmt.Println("Отправлено", string(b))
	fmt.Fprintln(w, string(b))
}

// apigen:api {"url": "/user/create", "auth": true, "method": "POST"}
func (srv *MyApi) Create(ctx context.Context, in CreateParams) (*NewUser, error) {

	if in.Login == "bad_username" {
		return nil, fmt.Errorf("bad user")
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()

	_, exist := srv.users[in.Login]
	if exist {
		return nil, ApiError{http.StatusConflict, fmt.Errorf("user %s exist", in.Login)}
	}

	id := srv.nextID
	srv.nextID++
	srv.users[in.Login] = &User{
		ID:       id,
		Login:    in.Login,
		FullName: in.Name,
		Status:   srv.statuses[in.Status],
	}

	return &NewUser{id}, nil
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
	// fmt.Println("Отправлено", string(b))
	fmt.Fprintln(w, string(b))
}

func main() {
	ts := httptest.NewServer(NewMyApi())

	cases := []Case{
		Case{ // успешный запрос
			Path:   ApiUserProfile,
			Query:  "login=rvasily",
			Status: http.StatusOK,
			Result: CR{
				"error": "",
				"response": CR{
					"id":        42,
					"login":     "rvasily",
					"full_name": "Vasily Romanov",
					"status":    20,
				},
			},
		},
		Case{ // успешный запрос - POST
			Path:   ApiUserProfile,
			Method: http.MethodPost,
			Query:  "login=rvasily",
			Status: http.StatusOK,
			Result: CR{
				"error": "",
				"response": CR{
					"id":        42,
					"login":     "rvasily",
					"full_name": "Vasily Romanov",
					"status":    20,
				},
			},
		},
		Case{ // сработала валидация - логин не должен быть пустым
			Path:   ApiUserProfile,
			Query:  "",
			Status: http.StatusBadRequest,
			Result: CR{
				"error": "login must me not empty",
			},
		},
		Case{ // получили ошибку общего назначения - ваш код сам подставил 500
			Path:   ApiUserProfile,
			Query:  "login=bad_user",
			Status: http.StatusInternalServerError,
			Result: CR{
				"error": "bad user",
			},
		},
		Case{ // получили специализированную ошибку - ваш код поставил статус 404 оттуда
			Path:   ApiUserProfile,
			Query:  "login=not_exist_user",
			Status: http.StatusNotFound,
			Result: CR{
				"error": "user not exist",
			},
		},
		Case{ // это должен ответить ваш ServeHTTP - если ему пришло что-то неизвестное (например когда он обрабатывает /user/)
			Path:   "/user/unknown",
			Query:  "login=not_exist_user",
			Status: http.StatusNotFound,
			Result: CR{
				"error": "unknown method",
			},
		},
		Case{ // создаём юзера
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=mr.moderator&age=32&status=moderator&full_name=Ivan_Ivanov",
			Status: http.StatusOK,
			Auth:   true,
			Result: CR{
				"error": "",
				"response": CR{
					"id": 43,
				},
			},
		},
		Case{ // юзер действительно создался
			Path:   ApiUserProfile,
			Query:  "login=mr.moderator",
			Status: http.StatusOK,
			Result: CR{
				"error": "",
				"response": CR{
					"id":        43,
					"login":     "mr.moderator",
					"full_name": "Ivan_Ivanov",
					"status":    10,
				},
			},
		},
		Case{ // только POST
			Path:   ApiUserCreate,
			Method: http.MethodGet,
			Query:  "login=mr.moderator&age=32&status=moderator&full_name=GetMethod",
			Status: http.StatusNotAcceptable,
			Auth:   true,
			Result: CR{
				"error": "bad method",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "any_params=123",
			Status: http.StatusForbidden,
			Auth:   false,
			Result: CR{
				"error": "unauthorized",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=mr.moderator&age=32&status=moderator&full_name=New_Ivan",
			Status: http.StatusConflict,
			Auth:   true,
			Result: CR{
				"error": "user mr.moderator exist",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "&age=32&status=moderator&full_name=Ivan_Ivanov",
			Status: http.StatusBadRequest,
			Auth:   true,
			Result: CR{
				"error": "login must me not empty",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=new_m&age=32&status=moderator&full_name=Ivan_Ivanov",
			Status: http.StatusBadRequest,
			Auth:   true,
			Result: CR{
				"error": "login len must be >= 10",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=new_moderator&age=ten&status=moderator&full_name=Ivan_Ivanov",
			Status: http.StatusBadRequest,
			Auth:   true,
			Result: CR{
				"error": "age must be int",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=new_moderator&age=-1&status=moderator&full_name=Ivan_Ivanov",
			Status: http.StatusBadRequest,
			Auth:   true,
			Result: CR{
				"error": "age must be >= 0",
			},
		},
		Case{
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=new_moderator&age=256&status=moderator&full_name=Ivan_Ivanov",
			Status: http.StatusBadRequest,
			Auth:   true,
			Result: CR{
				"error": "age must be <= 128",
			},
		},
		Case{ // status по-умолчанию
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=new_moderator3&age=32&full_name=Ivan_Ivanov",
			Status: http.StatusOK,
			Auth:   true,
			Result: CR{
				"error": "",
				"response": CR{
					"id": 44,
				},
			},
		},
		Case{ // обрабатываем неизвестную ошибку
			Path:   ApiUserCreate,
			Method: http.MethodPost,
			Query:  "login=bad_username&age=32&full_name=Ivan_Ivanov",
			Status: http.StatusInternalServerError,
			Auth:   true,
			Result: CR{
				"error": "bad user",
			},
		},
	}

	runTests(ts, cases)
}

func runTests(ts *httptest.Server, cases []Case) {
	for idx, item := range cases {
		var (
			err      error
			result   interface{}
			expected interface{}
			req      *http.Request
		)
		caseName := fmt.Sprintf("case %d: [%s] %s %s", idx, item.Method, item.Path, item.Query)

		if item.Method == http.MethodPost {
			reqBody := strings.NewReader(item.Query)
			req, err = http.NewRequest(item.Method, ts.URL+item.Path, reqBody)
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		} else {
			req, err = http.NewRequest(item.Method, ts.URL+item.Path+"?"+item.Query, nil)
		}

		if item.Auth {
			req.Header.Add("X-Auth", "100500")
		}

		resp, err := client.Do(req)
		// fmt.Println("RESPONSE", resp)
		if err != nil {
			fmt.Printf("[%s] request error: %v", caseName, err)
			continue
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		fmt.Println("RESPONSE BODY", string(body))
		fmt.Println()
		fmt.Println()
		err = json.Unmarshal(body, &result)
		if err != nil {
			fmt.Printf("[%s] cant unpack json: %v", caseName, err)
			continue
		}

		// reflect.DeepEqual не работает если нам приходят разные типы
		// а там приходят разные типы (string VS interface{}) по сравнению с тем что в ожидаемом результате
		// этот маленький грязный хак конвертит данные сначала в json, а потом обратно в interface - получаем совместимые результаты
		// не используйте это в продакшен-коде - надо явно писать что ожидается интерфейс или использовать другой подход с точным форматом ответа
		data, err := json.Marshal(item.Result)
		json.Unmarshal(data, &expected)

		if !reflect.DeepEqual(result, expected) {
			fmt.Printf("[%d] results not match\nGot: %#v\nExpected: %#v", idx, result, item.Result)
			continue
		} else {
			fmt.Printf("[%d] results match\nGot: %#v\nExpected: %#v", idx, result, item.Result)
		}
	}
}
