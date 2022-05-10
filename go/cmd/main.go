package main

import (
	"database/sql"
	"encoding/gob"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"os"
)

type key int

const (
	userKey key = iota
)

const templatesDir = "./templates/"

func main() {
	gob.Register(User{})
	gob.Register(userKey)

	host := os.Getenv("MYSQL_HOST")
	port := os.Getenv("MYSQL_PORT")
	user := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")
	database := os.Getenv("MYSQL_DATABASE")
	//host := "127.0.0.1"
	//port := "3306"
	//user := "stocktaking_app"
	//password := "stocktaking_app_pWd"
	//database := "stocktaking"

	DSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true", user, password, host, port, database)
	db, err := sql.Open("mysql", DSN)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	router, _ := NewRouter(db)

	fmt.Println("starting server at :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
