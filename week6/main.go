// тут лежит тестовый код
// менять вам может потребоваться только коннект к базе
package main

import (
	"database/sql"
	"fmt"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

var (
	// DSN это соединение с базой
	// вы можете изменить этот на тот который вам нужен
	// docker run -p 3306:3306 -v $(PWD):/docker-entrypoint-initdb.d -e MYSQL_ROOT_PASSWORD=1234 -e MYSQL_DATABASE=golang -d mysql
	// DSN = "root@tcp(localhost:3306)/golang2017?charset=utf8"
	DSN = "root:pass@tcp(localhost:3306)/golang?charset=utf8&interpolateParams=true"
	// DSN = "coursera:5QPbAUufx7@tcp(localhost:3306)/coursera?charset=utf8"
)

// func PrepareTestApis(db *sql.DB) {
// 	qs := []string{
// 		`DROP TABLE IF EXISTS items;`,

// 		`CREATE TABLE items (
//   id int(11) NOT NULL AUTO_INCREMENT,
//   title varchar(255) NOT NULL,
//   description text NOT NULL,
//   updated varchar(255) DEFAULT NULL,
//   PRIMARY KEY (id)
// ) ENGINE=InnoDB DEFAULT CHARSET=utf8;`,

// 		`INSERT INTO items (id, title, description, updated) VALUES
// (1,	'database/sql',	'Рассказать про базы данных',	'rvasily'),
// (2,	'memcache',	'Рассказать про мемкеш с примером использования',	NULL);`,

// 		`DROP TABLE IF EXISTS users;`,

// 		`CREATE TABLE users (
// 			user_id int(11) NOT NULL AUTO_INCREMENT,
//   login varchar(255) NOT NULL,
//   password varchar(255) NOT NULL,
//   email varchar(255) NOT NULL,
//   info text NOT NULL,
//   updated varchar(255) DEFAULT NULL,
//   PRIMARY KEY (user_id)
// ) ENGINE=InnoDB DEFAULT CHARSET=utf8;`,

// 		`INSERT INTO users (user_id, login, password, email, info, updated) VALUES
// (1,	'rvasily',	'love',	'rvasily@example.com',	'none',	NULL);`,
// 	}

// 	for _, q := range qs {
// 		_, err := db.Exec(q)
// 		if err != nil {
// 			panic(err)
// 		}
// 	}
// }

// func CleanupTestApis(db *sql.DB) {
// 	qs := []string{
// 		`DROP TABLE IF EXISTS items;`,
// 		`DROP TABLE IF EXISTS users;`,
// 	}
// 	for _, q := range qs {
// 		_, err := db.Exec(q)
// 		if err != nil {
// 			panic(err)
// 		}
// 	}
// }

// docker run --name test-mysql-3 -v D:\Code\Coursera\Go\week6lec\mysql:/docker-entrypoint-initdb.d -p 3306:3306 -e MYSQL_ROOT_PASSWORD=pass -e MYSQL_DATABASE=golang -d mysql
// в SQL powershell
// ALTER USER 'root' IDENTIFIED WITH mysql_native_password BY 'pass';
// flush privileges;

func main() {
	db, err := sql.Open("mysql", DSN)
	if err != nil {
		fmt.Println("mysql connection error")
		return
	}
	err = db.Ping() // вот тут будет первое подключение к базе
	if err != nil {
		panic(err)
	}
	// PrepareTestApis(db)
	// defer CleanupTestApis(db)

	handler, err := NewDbExplorer(db)
	if err != nil {
		panic(err)
	}

	fmt.Println("starting server at :8082")
	http.ListenAndServe(":8082", handler)
}
