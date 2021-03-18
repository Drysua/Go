package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
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

func MustAtoi(s string) int {
	res, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return res
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
	fmt.Printf("%+v\n", sr)

	xmlFile, err := os.Open(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	if !(sr.OrderField == "" || sr.OrderField == "Name" || sr.OrderField == "Age" || sr.OrderField == "Id") {
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

	// for i := 0; i < len(result); i++ {
	// 	fmt.Println("User Id: ", result[i].Id)
	// 	fmt.Println("User Name: " + result[i].Name)
	// 	fmt.Println("User Age: ", result[i].Age)
	// 	fmt.Print("User About: " + result[i].About)
	// 	fmt.Println("User Gender: " + result[i].Gender)
	// }
	// fmt.Println(len(result))

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
	fmt.Fprintln(w, string(b))

}

type TestCase struct {
	Request *SearchRequest
	Result  *SearchResponse
	IsError bool
}

type TestServer struct {
	server *httptest.Server
	Search SearchClient
}

func (ts *TestServer) Close() {
	ts.server.Close()
}

func newTestServer(token string) TestServer {
	server := httptest.NewServer(http.HandlerFunc(handler))
	client := SearchClient{token, server.URL}

	return TestServer{server, client}
}

func TestLimitLow(t *testing.T) {
	ts := newTestServer(AccessToken)
	defer ts.Close()

	_, err := ts.Search.FindUsers(SearchRequest{
		Limit: -1,
	})

	if err == nil {
		t.Errorf("Empty error")
	} else if err.Error() != "limit must be > 0" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestLimitHigh(t *testing.T) {
	ts := newTestServer(AccessToken)
	defer ts.Close()

	response, _ := ts.Search.FindUsers(SearchRequest{
		Limit: 100,
	})

	if len(response.Users) != 25 {
		t.Errorf("Invalid number of users: %d", len(response.Users))
	}
}

func TestInvalidToken(t *testing.T) {
	ts := newTestServer(AccessToken + "invalid")
	defer ts.Close()

	_, err := ts.Search.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if err.Error() != "Bad AccessToken" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestInvalidOrderField(t *testing.T) {
	ts := newTestServer(AccessToken)
	defer ts.Close()

	_, err := ts.Search.FindUsers(SearchRequest{
		OrderBy:    -1,
		OrderField: "Foo",
	})

	if err == nil {
		t.Errorf("Empty error")
	} else if err.Error() != "OrderFeld Foo invalid" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestOffsetLow(t *testing.T) {
	ts := newTestServer(AccessToken)
	defer ts.Close()

	_, err := ts.Search.FindUsers(SearchRequest{
		Offset: -1,
	})

	if err == nil {
		t.Errorf("Empty error")
	} else if err.Error() != "offset must be > 0" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestFindUserByName(t *testing.T) {
	ts := newTestServer(AccessToken)
	defer ts.Close()

	response, _ := ts.Search.FindUsers(SearchRequest{
		Query: "Annie",
		Limit: 1,
	})

	if len(response.Users) != 1 {
		t.Errorf("Invalid number of users: %d", len(response.Users))
		return
	}

	if response.Users[0].Name != "Annie Osborn" {
		t.Errorf("Invalid user found: %v", response.Users[0])
		return
	}
}

func TestLimitOffset(t *testing.T) {
	ts := newTestServer(AccessToken)
	defer ts.Close()

	response, _ := ts.Search.FindUsers(SearchRequest{
		Limit:  3,
		Offset: 0,
	})

	if len(response.Users) != 3 {
		t.Errorf("Invalid number of users: %d", len(response.Users))
		return
	}

	if response.Users[2].Name != "Brooks Aguilar" {
		t.Errorf("Invalid user at position 3: %v", response.Users[2])
		return
	}

	response, _ = ts.Search.FindUsers(SearchRequest{
		Limit:  5,
		Offset: 2,
	})

	if len(response.Users) != 5 {
		t.Errorf("Invalid number of users: %d", len(response.Users))
		return
	}

	if response.Users[0].Name != "Brooks Aguilar" {
		t.Errorf("Invalid user at position 3: %v", response.Users[0])
		return
	}
}

func TestFatalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Fatal Error", http.StatusInternalServerError)
	}))
	client := SearchClient{AccessToken, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if err.Error() != "SearchServer fatal error" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestCantUnpackError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Some Error", http.StatusBadRequest)
	}))
	client := SearchClient{AccessToken, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "cant unpack error json") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestUnknownBadRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sendError(w, "Unknown Error", http.StatusBadRequest)
	}))
	client := SearchClient{AccessToken, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "unknown bad request error") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestCantUnpackResultError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "None")
	}))
	client := SearchClient{AccessToken, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "cant unpack result json") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	client := SearchClient{AccessToken, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "timeout for") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestUnknownError(t *testing.T) {
	client := SearchClient{AccessToken, "error"}

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "unknown error") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}
