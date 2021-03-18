package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	_ "github.com/go-sql-driver/mysql"
)

type Item struct {
	Id          int
	Title       string
	Description string
	Updated     sql.NullString
}

type Handler struct {
	DB   *sql.DB
	Tmpl *template.Template
}

func MustAtoi(s string) int {
	res, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return res
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
			fmt.Println("Rows scan", err.Error())
			return items, err
		}

		for i := range row {
			if row[i] != nil {
				row[i] = string(row[i].([]byte))
			}
		}
		items = append(items, row)
	}
	return items, nil
}

func GetId(h *Handler, table string) (string, error) {
	// rows, err := h.DB.Query("SELECT COLUMN_NAME FROM information_schema.KEY_COLUMN_USAGE WHERE TABLE_NAME = ? AND CONSTRAINT_NAME = 'PRIMARY'", table)
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

// GET / - возвращает список все таблиц (которые мы можем использовать в дальнейших запросах)
// GET /$table?limit=5&offset=7 - возвращает список из 5 записей (limit) начиная с 7-й (offset) из таблицы $table. limit по-умолчанию 5, offset 0
// GET /$table/$id - возвращает информацию о самой записи или 404

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var rows *sql.Rows
	var err error
	fmt.Println("Q is ", q)
	fmt.Println("path: ", r.URL.Path)

	if len(q) == 0 {
		dir, base := path.Split(r.URL.Path)
		if dir == "/" {
			fmt.Println("Received /")
			rows, err = h.DB.Query("SHOW TABLES")
			if err != nil {
				fmt.Println("SHOW TABLES error", err.Error())
				return
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

		}
	} else {
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
			return
		}
		off, err := strconv.Atoi(offset)
		if err != nil {
			fmt.Println("offset is not a number: ", offset, err)
			return
		}

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
	}

	items, err := GetItems(rows)
	b, _ := json.Marshal(items)
	fmt.Println(string(b))
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
	// fmt.Println(cols)
	// fmt.Println(params)
	// fmt.Println(req)

	result, err := h.DB.Exec(req, params...)
	if err != nil {
		fmt.Println("PUT ERROR :", err)
		return
	}

	affected, err := result.RowsAffected()
	__err_panic(err)
	lastID, err := result.LastInsertId()
	__err_panic(err)

	fmt.Println("Insert - RowsAffected", affected, "LastInsertId: ", lastID)
	b, _ := json.Marshal(lastID)
	fmt.Println(string(b))
	fmt.Fprintln(w, string(b))
}

// POST /$table/$id - обновляет запись, данные приходят в теле запроса (POST-параметры)
func (h *Handler) Post(w http.ResponseWriter, r *http.Request) {
	dir, base := path.Split(r.URL.Path)
	table := "`" + dir[1:len(dir)-1] + "`"

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
		return
	}

	req := ""
	for i, col := range cols {
		req += col + " = " + placeholders[i] + ","
	}
	params = append(params, id)
	result, err := h.DB.Exec("UPDATE "+table+" SET "+req[:len(req)-1]+" WHERE "+table_id+" = ?", params...)
	if err != nil {
		fmt.Println(cols)
		fmt.Println(params)
		fmt.Println("UPDATE " + table + " SET " + req[:len(req)-1] + " WHERE " + table_id + " = ?")
		fmt.Println("Post exec error: ", err)
		return
	}
	affected, err := result.RowsAffected()
	fmt.Println("Post -- RowsAffected", affected)
	b, _ := json.Marshal(affected)
	fmt.Println(string(b))
	fmt.Fprintln(w, string(b))

}

// DELETE /$table/$id - удаляет запись
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	dir, base := path.Split(r.URL.Path)
	table := "`" + dir[1:len(dir)-1] + "`"

	table_id, err := GetId(h, table)

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

	affected, err := result.RowsAffected()
	__err_panic(err)

	fmt.Println("Delete - RowsAffected", affected)

	w.Header().Set("Content-type", "application/json")
	resp := `{"affected": ` + strconv.Itoa(int(affected)) + `}`
	w.Write([]byte(resp))
}

func CleanupDB(db *sql.DB) {
	qs := []string{
		`DROP TABLE IF EXISTS items;`,
		`DROP TABLE IF EXISTS users;`,
	}
	for _, q := range qs {
		_, err := db.Exec(q)
		if err != nil {
			panic(err)
		}
	}
}

func (h *Handler) Clean(w http.ResponseWriter, r *http.Request) {
	CleanupDB(h.DB)
}

func (h *Handler) AddForm(w http.ResponseWriter, r *http.Request) {
	err := h.Tmpl.ExecuteTemplate(w, "create.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DELETE /$table/$id - удаляет запись

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	__err_panic(err)

	// в целям упрощения примера пропущена валидация
	result, err := h.DB.Exec(
		"UPDATE items SET"+
			"`title` = ?"+
			",`description` = ?"+
			",`updated` = ?"+
			"WHERE id = ?",
		r.FormValue("title"),
		r.FormValue("description"),
		"rvasily",
		id,
	)
	__err_panic(err)

	affected, err := result.RowsAffected()
	__err_panic(err)

	fmt.Println("Update - RowsAffected", affected)

	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) Del(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	__err_panic(err)

	result, err := h.DB.Exec(
		"DELETE FROM items WHERE id = ?",
		id,
	)
	__err_panic(err)

	affected, err := result.RowsAffected()
	__err_panic(err)

	fmt.Println("Delete - RowsAffected", affected)

	w.Header().Set("Content-type", "application/json")
	resp := `{"affected": ` + strconv.Itoa(int(affected)) + `}`
	w.Write([]byte(resp))
}

func MakeDB(db *sql.DB) {
	qs := []string{
		`DROP TABLE IF EXISTS items;`,

		`CREATE TABLE items (
  id int(11) NOT NULL AUTO_INCREMENT,
  title varchar(255) NOT NULL,
  description text NOT NULL,
  updated varchar(255) DEFAULT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;`,

		`INSERT INTO items (id, title, description, updated) VALUES
(1,	'database/sql',	'Рассказать про базы данных',	'rvasily'),
(2,	'memcache',	'Рассказать про мемкеш с примером использования',	NULL);`,

		`DROP TABLE IF EXISTS users;`,

		`CREATE TABLE users (
			user_id int(11) NOT NULL AUTO_INCREMENT,
  login varchar(255) NOT NULL,
  password varchar(255) NOT NULL,
  email varchar(255) NOT NULL,
  info text NOT NULL,
  updated varchar(255) DEFAULT NULL,
  PRIMARY KEY (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;`,

		`INSERT INTO users (user_id, login, password, email, info, updated) VALUES
(1,	'rvasily',	'love',	'rvasily@example.com',	'none',	NULL);`,
	}
	for _, q := range qs {
		_, err := db.Exec(q)
		if err != nil {
			panic(err)
		}
	}
}

func main() {
	// docker run --name test-mysql-3 -v D:\Code\Coursera\Go\week6lec\mysql:/docker-entrypoint-initdb.d -p 3306:3306 -e MYSQL_ROOT_PASSWORD=pass -e MYSQL_DATABASE=golang -d mysql
	// основные настройки к базе
	dsn := "root:pass@tcp(localhost:3306)/golang?"
	// указываем кодировку
	dsn += "&charset=utf8"
	// отказываемся от prapared statements
	// параметры подставляются сразу
	dsn += "&interpolateParams=true"

	db, err := sql.Open("mysql", dsn)

	db.SetMaxOpenConns(10)

	err = db.Ping() // вот тут будет первое подключение к базе
	if err != nil {
		panic(err)
	}
	MakeDB(db)

	handlers := &Handler{
		DB:   db,
		Tmpl: template.Must(template.ParseGlob("../crud_templates/*")),
	}

	// в целям упрощения примера пропущена авторизация и csrf
	r := mux.NewRouter()
	// r.HandleFunc("/", handlers.List).Methods("GET")
	// r.HandleFunc("/items", handlers.List).Methods("GET")
	// r.HandleFunc("/items/new", handlers.AddForm).Methods("GET")
	// r.HandleFunc("/items/new", handlers.Add).Methods("POST")
	// r.HandleFunc("/items/{id}", handlers.Edit).Methods("GET")
	// r.HandleFunc("/items/{id}", handlers.Update).Methods("POST")
	// r.HandleFunc("/items/{id}", handlers.Delete).Methods("DELETE")
	r.HandleFunc("/", handlers.Get).Methods("GET")
	r.HandleFunc("/clean", handlers.Clean).Methods("GET")
	r.HandleFunc("/items/{id}", handlers.Get).Methods("GET")
	r.HandleFunc("/items", handlers.Get).Methods("GET")
	r.HandleFunc("/items/", handlers.Put).Methods("PUT")
	r.HandleFunc("/users/{id}", handlers.Get).Methods("GET")
	r.HandleFunc("/users", handlers.Get).Methods("GET")
	r.HandleFunc("/users/", handlers.Put).Methods("PUT")
	r.HandleFunc("/items/{id}", handlers.Post).Methods("POST")
	r.HandleFunc("/users/{id}", handlers.Post).Methods("POST")
	r.HandleFunc("/items/{id}", handlers.Delete).Methods("DELETE")
	r.HandleFunc("/users/{id}", handlers.Delete).Methods("DELETE")
	fmt.Println("starting server at :8080")
	http.ListenAndServe(":8080", r)
}

// не используйте такой код в прошакшене
// ошибка должна всегда явно обрабатываться
func __err_panic(err error) {
	if err != nil {
		panic(err)
	}
}
