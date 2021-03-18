package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
)

const filePath string = "dataset.xml"
const AccessToken = "1234567asdfg"

type UserXML struct {
	XMLName   xml.Name `xml:"row" json:"-"`
	Id        int      `xml:"id" json:"id"`
	FirstName string   `xml:"first_name" json:"first_name"`
	LastName  string   `xml:"last_name" json:"last_name"`
	Age       int      `xml:"age" json:"age"`
	About     string   `xml:"about" json:"about"`
	Gender    string   `xml:"gender" json:"gender"`
}

type Users struct {
	XMLName xml.Name  `xml:"root"`
	Users   []UserXML `xml:"row"`
}

type User struct {
	Id     int
	Name   string
	Age    int
	About  string
	Gender string
}

type SearchRequest struct {
	Limit      int
	Offset     int    // Можно учесть после сортировки
	Query      string // подстрока в 1 из полей
	OrderField string
	OrderBy    int // -1 по убыванию, 0 как встретилось, 1 по возрастанию
}

type SearchErrorResponse struct {
	Error string
}

func sendError(w http.ResponseWriter, error string, code int) {
	js, err := json.Marshal(SearchErrorResponse{error})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintln(w, string(js))
}

type CreateParams struct {
	Login  string `apivalidator:"required,min=10"`
	Name   string `apivalidator:"paramname=full_name"`
	Status string `apivalidator:"enum=user|moderator|admin,default=user"`
	Age    int    `apivalidator:"min=0,max=128"`
}

func MustAtoi(s string) int {
	res, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return res
}

func GetParams(r *http.Request) CreateParams {
	var params CreateParams
	if r.Method == "GET" {
		q := r.URL.Query()
		params := CreateParams{
			Login:  q["login"][0],
			Name:   q["name"][0],
			Status: q["status"][0],
			Age:    MustAtoi(q["age"][0]),
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
		params := CreateParams{
			Login:  q.Get("login"),
			Name:   q.Get("name"),
			Status: q.Get("status"),
			Age:    MustAtoi(q.Get("age")),
		}
	}
	fmt.Println(params)
	return params
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("AccessToken") != AccessToken {
		http.Error(w, "Invalid access token", http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	sr := SearchRequest{
		Limit:      MustAtoi(q["limit"][0]),
		Offset:     MustAtoi(q["offset"][0]),
		Query:      strings.ToLower(q["query"][0]),
		OrderField: q["order_field"][0],
		OrderBy:    MustAtoi(q["order_by"][0]),
	}
	fmt.Printf("Получено %+v\n", sr)

	xmlFile, err := os.Open(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Println("XML file open error")
		return
	}
	defer xmlFile.Close()
	data, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var users Users
	result := []User{}

	err = xml.Unmarshal(data, &users)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// fmt.Println(users)
	for i := 0; i < len(users.Users); i++ {
		if sr.Query != "" {
			str := strings.ToLower(users.Users[i].FirstName + " " + users.Users[i].LastName + " " + users.Users[i].About + " " + users.Users[i].Gender)
			if strings.Contains(str, sr.Query) {
				result = append(result, User{
					Id:     users.Users[i].Id,
					Name:   users.Users[i].FirstName + " " + users.Users[i].LastName,
					Age:    users.Users[i].Age,
					About:  users.Users[i].About,
					Gender: users.Users[i].Gender,
				})
			}

		} else {
			// fmt.Println("here", i)
			result = append(result, User{
				Id:     users.Users[i].Id,
				Name:   users.Users[i].FirstName + " " + users.Users[i].LastName,
				Age:    users.Users[i].Age,
				About:  users.Users[i].About,
				Gender: users.Users[i].Gender,
			})
		}
	}
	if !(sr.OrderBy == -1 || sr.OrderBy == 0 || sr.OrderBy == 1) {
		sendError(w, "ErrorBadOrderField", http.StatusBadRequest)
		return
	}
	if sr.OrderBy != 0 {
		if sr.OrderField == "Id" {
			sort.Slice(result[:], func(i, j int) bool {
				if sr.OrderBy == -1 {
					return result[i].Id > result[j].Id
				}
				return result[i].Id < result[j].Id

			})
		} else if sr.OrderField == "Age" {
			sort.Slice(result[:], func(i, j int) bool {
				if sr.OrderBy == -1 {
					return result[i].Age > result[j].Age
				}
				return result[i].Age < result[j].Age
			})
		} else {
			sort.Slice(result[:], func(i, j int) bool {
				if sr.OrderBy == -1 {
					return result[i].Name[0] < result[j].Name[0]
				}
				return result[i].Name[0] < result[j].Name[0]
			})
		}
	}
	if sr.Offset > len(result) {
		result = []User{}
	} else {
		if sr.Offset+sr.Limit > len(result) {
			result = result[sr.Offset:]
		} else {
			result = result[sr.Offset : sr.Offset+sr.Limit]
		}
	}

	for i := 0; i < len(result); i++ {
		fmt.Println("User Id: ", result[i].Id)
		fmt.Println("User Name: " + result[i].Name)
		fmt.Println("User Age: ", result[i].Age)
		fmt.Print("User About: " + result[i].About)
		fmt.Println("User Gender: " + result[i].Gender)
	}
	fmt.Println(len(result))

	// // header := r.Header
	// fmt.Println(reflect.TypeOf(q))
	// fmt.Println(q)
	b, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	// w.Write(b)
	fmt.Println("Отправлено", result)
	fmt.Fprintln(w, string(b))

}

func GetParams(r *http.Request) CreateParams {
	var params CreateParams
	if r.Method == "GET" {
		q := r.URL.Query()
		params := CreateParams{
			Login:  q["login"][0],
			Name:   q["name"][0],
			Status: q["status"][0],
			Age:    MustAtoi(q["age"][0]),
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
		params := CreateParams{
			Login:  q.Get("login"),
			Name:   q.Get("name"),
			Status: q.Get("status"),
			Age:    MustAtoi(q.Get("age")),
		}
	}
	fmt.Println(params)
	return params
}

func handler1(w http.ResponseWriter, r *http.Request) {
	params := GetParams(r)
	fmt.Printf("Получено %+v\n", params)
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
func main() {
	http.HandleFunc("/", handler1)

	fmt.Println("starting server at :8080")
	http.ListenAndServe(":8080", nil)
}
