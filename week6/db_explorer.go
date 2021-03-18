package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"reflect"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type Handler struct {
	DB     *sql.DB
	Tables []string
}

func NewDbExplorer(db *sql.DB) (*Handler, error) {
	tables := GetTables(db)
	return &Handler{
		DB:     db,
		Tables: tables,
	}, nil
}

func GetTables(db *sql.DB) []string {
	rows, _ := db.Query("SHOW TABLES")
	items, _ := GetItems(rows)
	fmt.Println("GetTables", items)
	var result []string
	for _, item := range items {
		result = append(result, item[0].(string))
	}
	return result

}

func sendError(w http.ResponseWriter, error string, code int) {
	js, err := json.Marshal(map[string]interface{}{"error": error})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintln(w, string(js))
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		h.Get(w, r)
	case "PUT":
		h.Put(w, r)
	case "POST":
		h.Post(w, r)
	case "DELETE":
		h.Delete(w, r)
	default:
		// sendError(w, "unknown method", http.StatusNotFound)
		fmt.Println("unkown method")
	}
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

//get table items from row
func GetItems(rows *sql.Rows) ([][]interface{}, error) {

	items := make([][]interface{}, 0)

	defer rows.Close()
	types, _ := rows.ColumnTypes()
	for rows.Next() {
		row := make([]interface{}, len(types))
		for i := range row {
			row[i] = &row[i]
		}
		err := rows.Scan(row...)
		if err != nil {
			return items, err
		}

		for i := range row {
			if row[i] != nil {
				switch row[i].(type) {
				case []byte:
					row[i] = string(row[i].([]byte))
				case int64:
					row[i] = fmt.Sprint(row[i].(int64))
				default:
					fmt.Println("unknown type")
				}

			}
		}
		items = append(items, row)
	}
	return items, nil
}

func GetId(h *Handler, table string) (string, error) {
	rows, err := h.DB.Query("SHOW KEYS FROM " + table + " WHERE Key_name = 'PRIMARY'")
	if err != nil {
		return "", err
	}
	items, _ := GetItems(rows)
	// fmt.Println(items[0][4].(string))
	return items[0][4].(string), nil
}

func GetTableSize(h *Handler, table string) (int, error) {
	rows, err := h.DB.Query("SELECT COUNT(1) FROM " + table)
	if err != nil {
		return 0, err
	}
	items, _ := GetItems(rows)
	result, _ := strconv.Atoi(items[0][0].(string))
	return result, nil
}

func GetResult(rows *sql.Rows) []map[string]interface{} {
	columns, _ := rows.Columns()
	colTypes, _ := rows.ColumnTypes()
	items, _ := GetItems(rows)
	fmt.Println("GET RESULT", items)
	fmt.Println("GET RESULT", columns)
	var set []map[string]interface{}
	for i := range items {
		item := make(map[string]interface{}, len(columns))
		for j := range columns {
			// fmt.Println(colTypes[j].DatabaseTypeName())
			item[columns[j]] = items[i][j]
			if colTypes[j].DatabaseTypeName()[:3] == "INT" {
				item[columns[j]], _ = strconv.Atoi(items[i][j].(string))
			}
		}
		set = append(set, item)
	}
	fmt.Println("SET", set)
	return set
}

// GET / - возвращает список все таблиц (которые мы можем использовать в дальнейших запросах)
// GET /$table?limit=5&offset=7 - возвращает список из 5 записей (limit) начиная с 7-й (offset) из таблицы $table. limit по-умолчанию 5, offset 0
// GET /$table/$id - возвращает информацию о самой записи или 404
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var rows *sql.Rows
	result := make(map[string]interface{}, 1)
	dir, base := path.Split(r.URL.Path)
	fmt.Println(r.URL)
	if dir == "/" {
		if base != "" {
			if !contains(h.Tables, base) {
				sendError(w, "unknown table", http.StatusNotFound)
				return
			}
			table := r.URL.Path[1:]
			limit := q.Get("limit")
			offset := q.Get("offset")

			if q.Get("limit") == "" {
				limit = "5"
			}
			if q.Get("offset") == "" {
				offset = "0"
			}

			lim, err := strconv.Atoi(limit)
			if err != nil {
				fmt.Println("limit is not a number: ", limit, err)
				lim = 5
			}
			off, err := strconv.Atoi(offset)
			if err != nil {
				fmt.Println("offset is not a number: ", offset, err)
				off = 0
			}

			fmt.Println("HEREEEE", table, limit, offset)

			size, err := GetTableSize(h, table)
			if err != nil {
				fmt.Println("GetTableSize err: ", err)
			}
			if lim+off > size {
				lim = size
				off = 0
			}

			rows, err = h.DB.Query("SELECT * FROM "+table+" LIMIT ? OFFSET ?", lim, off)
			if err != nil {
				fmt.Println("limit/offset error", err.Error())
				return
			}
			result["records"] = GetResult(rows)

		} else {
			result["tables"] = h.Tables
		}
	} else {
		table := "`" + dir[1:len(dir)-1] + "`"
		id, err := strconv.Atoi(base)
		if err != nil {
			fmt.Println("GET WRONG ID: ", err)
			return
		}
		table_id, err := GetId(h, table)
		if err != nil {
			fmt.Println("GetId rows.Scan error: ", err)
			return
		}
		rows, err = h.DB.Query("SELECT * FROM "+table+" WHERE "+table_id+" = ?", id)
		if err != nil {
			fmt.Println("id error", err.Error())
			return
		}
		res := GetResult(rows)
		if len(res) == 0 {
			sendError(w, "record not found", http.StatusNotFound)
			return
		}
		result["record"] = res[0]
	}

	response := map[string]interface{}{
		"response": result,
	}
	b, _ := json.Marshal(response)
	fmt.Fprintln(w, string(b))
}

func GetCols(h *Handler, table string, r *http.Request) ([]string, []interface{}, []string, error) {
	var cols []string
	var params []interface{}
	var placeholders []string
	rows, err := h.DB.Query("SHOW FULL COLUMNS FROM " + table)
	if err != nil {
		return cols, params, placeholders, err
	}
	items, err := GetItems(rows)
	if err != nil {
		fmt.Println("GetItems error: ", err)
		return cols, params, placeholders, err
	}

	if r.Header.Get("Content-Type") == "application/json" {
		reqBody, _ := ioutil.ReadAll(r.Body)
		got := make(map[string]interface{}, 1)
		fmt.Println("GOT ", got)
		err = json.Unmarshal(reqBody, &got)
		fmt.Println("got ", got)
		if err != nil {
			return cols, params, placeholders, err
		}
		if len(got) == 0 {
			return cols, params, placeholders, errors.New("empty request")
		}
		for _, str := range items {
			if got[str[0].(string)] != nil {
				t := reflect.TypeOf(got[str[0].(string)]).String()
				fmt.Println("GOT ", got[str[0].(string)])
				fmt.Println("REFLECT ", reflect.TypeOf(got[str[0].(string)]).String())
				b1 := t == "float64" && str[1].(string)[:3] == "int"
				b2 := t == "string" && (str[1].(string) == "text" || str[1].(string) == "varchar(255)")
				fmt.Println(str[1].(string))
				fmt.Println(b1, b2)
				if !(b1 || b2) {
					return cols, params, placeholders, errors.New("field " + str[0].(string) + " have invalid type")
				}
			} else {
				_, ok := got[str[0].(string)]
				if str[3].(string) == "NO" && ok {
					return cols, params, placeholders, errors.New("field " + str[0].(string) + " have invalid type")
				}
			}

			if str[4].(string) == "PRI" {
				if r.Method == "POST" && got[str[0].(string)] != nil {
					return cols, params, placeholders, errors.New("field " + str[0].(string) + " have invalid type")
				}
				continue
			}

			if str[3].(string) == "NO" {
				fmt.Println(str[0].(string), got[str[0].(string)], str[3].(string))
				if val, ok := got[str[0].(string)]; val == nil {
					if !ok && str[5] == nil && r.Method == "PUT" {
						cols = append(cols, "`"+str[0].(string)+"`")
						placeholders = append(placeholders, "?")
						if str[1].(string)[:3] == "int" {
							params = append(params, 0)
						}
						if str[1].(string) == "text" || str[1].(string) == "varchar(255)" {
							params = append(params, "")
						}
					}
					continue
				}
				cols = append(cols, "`"+str[0].(string)+"`")
				if got[str[0].(string)] == "" {
					placeholders = append(placeholders, "''")
				} else {
					placeholders = append(placeholders, "?")
					params = append(params, got[str[0].(string)])
				}
			} else {
				if got[str[0].(string)] == nil {
					params = append(params, sql.NullString{})
				} else {
					params = append(params, got[str[0].(string)])
				}
				cols = append(cols, "`"+str[0].(string)+"`")
				placeholders = append(placeholders, "?")
			}
		}
	} else {
		r.ParseForm()
		for _, str := range items {
			if str[4].(string) == "PRI" {
				continue
			}
			if str[3].(string) == "NO" {
				fmt.Println(str[0].(string), r.FormValue(str[0].(string)), str[3].(string))
				cols = append(cols, "`"+str[0].(string)+"`")
				if r.FormValue(str[0].(string)) == "" {
					placeholders = append(placeholders, "''")
				} else {
					placeholders = append(placeholders, "?")
					params = append(params, "'"+r.FormValue(str[0].(string))+"'")
				}
			}
		}
	}
	return cols, params, placeholders, nil
}

// PUT /$table - создаёт новую запись, данный по записи в теле запроса (POST-параметры)
func (h *Handler) Put(w http.ResponseWriter, r *http.Request) {
	// в целям упрощения примера пропущена валидация
	table := r.URL.Path[1 : len(r.URL.Path)-1]
	// fmt.Println(table)
	cols, params, placeholders, err := GetCols(h, table, r)
	if err != nil {
		fmt.Println("GetCols error: ", err)
		return
	}
	if err != nil {
		fmt.Println("show full columns error : ", err)
		return
	}
	req := "INSERT INTO " + table + " (" + strings.Join(cols, ",") + ") VALUES (" + strings.Join(placeholders, ",") + ")"
	result, err := h.DB.Exec(req, params...)
	if err != nil {
		fmt.Println(req)
		fmt.Println("PUT ERROR :", err)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		fmt.Println("Rows affected err: ", affected)
		return
	}
	lastID, err := result.LastInsertId()
	if err != nil {
		fmt.Println("Last id err: ", err)
		return
	}
	id, _ := GetId(h, table)

	fmt.Println("Insert - RowsAffected", affected, "LastInsertId: ", lastID)
	response := map[string]interface{}{
		"response": map[string]interface{}{
			id: lastID,
		},
	}

	b, _ := json.Marshal(response)
	fmt.Fprintln(w, string(b))
}

// POST /$table/$id - обновляет запись, данные приходят в теле запроса (POST-параметры)
func (h *Handler) Post(w http.ResponseWriter, r *http.Request) {
	dir, base := path.Split(r.URL.Path)
	table := "`" + dir[1:len(dir)-1] + "`"

	fmt.Println("Here POST ", r.URL.Path)
	table_id, err := GetId(h, table)
	if err != nil {
		fmt.Println("GetId rows.Scan error: ", err)
		return
	}
	id, err := strconv.Atoi(base)
	if err != nil {
		fmt.Println("POST WRONG ID: ", err)
		return
	}

	cols, params, placeholders, err := GetCols(h, table, r)
	if err != nil {
		fmt.Println("GetCols error: ", err)
		sendError(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Println(cols, params, placeholders)

	req := ""
	for i, col := range cols {
		req += col + " = " + placeholders[i] + ","
	}
	params = append(params, id)
	fmt.Println("UPDATE " + table + " SET " + req[:len(req)-1] + " WHERE " + table_id + " = ?")
	result, err := h.DB.Exec("UPDATE "+table+" SET "+req[:len(req)-1]+" WHERE "+table_id+" = ?", params...)
	if err != nil {
		fmt.Println(cols)
		fmt.Println(params)
		fmt.Println("UPDATE " + table + " SET " + req[:len(req)-1] + " WHERE " + table_id + " = ?")
		fmt.Println("Post exec error: ", err)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		fmt.Println("Affected error: ", err)
		return
	}
	fmt.Println("Post -- RowsAffected", affected)
	response := map[string]interface{}{
		"response": map[string]interface{}{
			"updated": affected,
		},
	}

	b, _ := json.Marshal(response)
	fmt.Fprintln(w, string(b))
}

// DELETE /$table/$id - удаляет запись
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	dir, base := path.Split(r.URL.Path)
	table := "`" + dir[1:len(dir)-1] + "`"

	table_id, err := GetId(h, table)
	if err != nil {
		fmt.Println("GetId error: ", err)
		return
	}

	id, err := strconv.Atoi(base)
	if err != nil {
		fmt.Println("DELETE WRONG ID: ", err)
		return
	}

	result, err := h.DB.Exec("DELETE FROM "+table+" WHERE "+table_id+" = ?", id)
	if err != nil {
		fmt.Println("DELETE ERROR: ", err)
		return
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		fmt.Println("Rows affected err: ", err)
		return
	}

	fmt.Println("Delete - RowsAffected", deleted)
	response := map[string]interface{}{
		"response": map[string]interface{}{
			"deleted": deleted,
		},
	}

	b, _ := json.Marshal(response)
	fmt.Fprintln(w, string(b))
}
