package main

import (
	"errors"
	"github.com/gorilla/sessions"
	"log"
	"net/http"
)

const SessionName = "session"

type LoginData struct {
	EmptyData          bool
	WrongLoginPassword bool
	User               *User
}

type User struct {
	ID              int64
	Login           string
	password        string
	AccessImport    bool
	AccessContract  bool
	AccessDeleteAct bool
	AccessReport    bool
	ActEmail        bool
}

func LoginProcess(w http.ResponseWriter, r *http.Request, router *Router) (*LoginData, int, error) {
	db := router.db
	cookie := router.cookie
	data := LoginData{}
	err := r.ParseForm()
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("error form parse")
	}
	login := r.Form.Get("login")
	password := r.Form.Get("password")
	if login == "" || password == "" {
		data.EmptyData = true
		return &data, http.StatusOK, nil
	}
	rows, err := db.Query("SELECT * FROM user WHERE login = ? and password = MD5(?) LIMIT 1", login, password)
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	exist := rows.Next()
	if !exist {
		data.WrongLoginPassword = true
		return &data, http.StatusOK, nil
	}
	user := User{}
	err = rows.Scan(&user.ID, &user.Login, &user.password, &user.AccessImport, &user.AccessContract, &user.AccessDeleteAct, &user.AccessReport, &user.ActEmail)
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	session, err := cookie.Get(r, SessionName)
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	session.Values["user"] = user
	session.Options.MaxAge = router.sessionLifetime
	err = sessions.Save(r, w)
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	data.User = &user
	return &data, 0, nil
}
