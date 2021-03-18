package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	orderAsc = iota
	orderDesc
)

var (
	errTest = errors.New("testing")
	client  = &http.Client{Timeout: time.Second}
)

type User struct {
	// XMLName   xml.Name `xml:"row" json:"-"`
	Id        int    `xml:"id" json:"id"`
	FirstName string `xml:"first_name" json:"first_name"`
	LastName  string `xml:"last_name" json:"last_name"`
	Age       int    `xml:"age" json:"age"`
	About     string `xml:"about" json:"about"`
	Gender    string `xml:"gender" json:"gender"`
}

type SearchResponse struct {
	Users []User
	// NextPage bool
}

type SearchErrorResponse struct {
	Error string
}

const (
	OrderByAsc  = -1
	OrderByAsIs = 0
	OrderByDesc = 1

	ErrorBadOrderField = `OrderField invalid`
)

type SearchRequest struct {
	Limit      int
	Offset     int    // Можно учесть после сортировки
	Query      string // подстрока в 1 из полей
	OrderField string
	// -1 по убыванию, 0 как встретилось, 1 по возрастанию
	OrderBy int
}

type SearchClient struct {
	// токен, по которому происходит авторизация на внешней системе, уходит туда через хедер
	AccessToken string
	// урл внешней системы, куда идти
	URL string
}

//FindUsers отправляет запрос во внешнюю систему, которая непосредственно ищет пользоваталей
func (srv *SearchClient) FindUsers(req SearchRequest) (*SearchResponse, error) {

	searcherParams := url.Values{}

	if req.Limit < 0 {
		return nil, fmt.Errorf("limit must be > 0")
	}
	if req.Limit > 25 {
		req.Limit = 25
	}
	if req.Offset < 0 {
		return nil, fmt.Errorf("offset must be > 0")
	}

	//нужно для получения следующей записи, на основе которой мы скажем - можно показать переключатель следующей страницы или нет
	req.Limit++

	searcherParams.Add("limit", strconv.Itoa(req.Limit))
	searcherParams.Add("offset", strconv.Itoa(req.Offset))
	searcherParams.Add("query", req.Query)
	searcherParams.Add("order_field", req.OrderField)
	searcherParams.Add("order_by", strconv.Itoa(req.OrderBy))

	searcherReq, err := http.NewRequest("GET", srv.URL+"?"+searcherParams.Encode(), nil)
	searcherReq.Header.Add("AccessToken", srv.AccessToken)

	resp, err := client.Do(searcherReq)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return nil, fmt.Errorf("timeout for %s", searcherParams.Encode())
		}
		return nil, fmt.Errorf("unknown error %s", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("Bad AccessToken")
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("SearchServer fatal error")
	case http.StatusBadRequest:
		errResp := SearchErrorResponse{}
		err = json.Unmarshal(body, &errResp)
		if err != nil {
			return nil, fmt.Errorf("cant unpack error json: %s", err)
		}
		if errResp.Error == "ErrorBadOrderField" {
			return nil, fmt.Errorf("OrderFeld %s invalid", req.OrderField)
		}
		return nil, fmt.Errorf("unknown bad request error: %s", errResp.Error)
	}

	data := []User{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("cant unpack result json: %s", err)
	}

	result := SearchResponse{}
	if len(data) == req.Limit {
		result.NextPage = true
		result.Users = data[0 : len(data)-1]
	} else {
		result.Users = data[0:len(data)]
	}

	return &result, err
}

func (srv *SearchClient) FindUsers1(req SearchRequest) {

	searcherParams := url.Values{}

	//нужно для получения следующей записи, на основе которой мы скажем - можно показать переключатель следующей страницы или нет
	req.Limit++
	searcherParams.Add("limit", strconv.Itoa(req.Limit))
	searcherParams.Add("offset", strconv.Itoa(req.Offset))
	searcherParams.Add("query", req.Query)
	searcherParams.Add("order_field", req.OrderField)
	searcherParams.Add("order_by", strconv.Itoa(req.OrderBy))

	searcherReq, _ := http.NewRequest("GET", srv.URL+"?"+searcherParams.Encode(), nil)
	fmt.Println(srv.URL + "?" + searcherParams.Encode())
	searcherReq.Header.Add("AccessToken", srv.AccessToken)

	resp, _ := client.Do(searcherReq)
	// resp, _ := http.Get(srv.URL + "?" + searcherParams.Encode())
	// if err != nil {
	// 	if err, ok := err.(net.Error); ok && err.Timeout() {
	// 		return nil, fmt.Errorf("timeout for %s", searcherParams.Encode())
	// 	}
	// 	return nil, fmt.Errorf("unknown error %s", err)
	// }

	defer resp.Body.Close()
	// body, _ := ioutil.ReadAll(resp.Body)

	// switch resp.StatusCode {
	// case http.StatusUnauthorized:
	// 	return nil, fmt.Errorf("Bad AccessToken")
	// case http.StatusInternalServerError:
	// 	return nil, fmt.Errorf("SearchServer fatal error")
	// case http.StatusBadRequest:
	// 	errResp := SearchErrorResponse{}
	// 	err = json.Unmarshal(body, &errResp)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("cant unpack error json: %s", err)
	// 	}
	// 	if errResp.Error == "ErrorBadOrderField" {
	// 		return nil, fmt.Errorf("OrderFeld %s invalid", req.OrderField)
	// 	}
	// 	return nil, fmt.Errorf("unknown bad request error: %s", errResp.Error)
	// }
	// var sr map[string][]string
	// err := json.NewDecoder(resp.Body).Decode(&sr)
	// fmt.Println(err)
	b, _ := ioutil.ReadAll(resp.Body)
	test := []User{}
	// err := json.NewDecoder(resp.Body).Decode(&test)

	err := json.Unmarshal(b, &test)
	if err != nil {
		fmt.Println(err)
	}

	// fmt.Printf("%+v", result)
	// fmt.Println(body[0])
	// fmt.Println(string(b))
	fmt.Printf("%+v", test)
}

func main() {
	sc := SearchClient{
		AccessToken: "1234567asdfg",
		URL:         "http://127.0.0.1:8080/",
	}
	sr := SearchRequest{
		Limit:      5,
		Offset:     0,        // Можно учесть после сортировки
		Query:      "female", // подстрока в 1 из полей
		OrderField: "Age",
		// -1 по убыванию, 0 как встретилось, 1 по возрастанию
		OrderBy: -1,
	}

	sc.FindUsers1(sr)
}
