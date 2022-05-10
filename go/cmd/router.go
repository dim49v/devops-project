package main

import (
	"context"
	"database/sql"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const defaultSessionLifetime = 7200

type MS map[string]interface{}

type Router struct {
	db              *sql.DB
	cookie          *sessions.CookieStore
	tmpl            *template.Template
	sessionLifetime int
	mux.Router
}

var pages = map[string]string{
	"login":      "Вход",
	"acts":       "Акты",
	"statistics": "Статистика",
}

func NewRouter(db *sql.DB) (http.Handler, error) {
	router := &Router{}
	router.db = db
	router.cookie = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
	sessionLifetime, err := strconv.Atoi(os.Getenv("SESSION_LIFETIME"))
	if err != nil {
		sessionLifetime = defaultSessionLifetime
	}
	router.sessionLifetime = sessionLifetime

	tmpl := template.New("base.gohtml")
	tmpl.Funcs(template.FuncMap{
		"nowYear": func() string {
			return time.Now().Format("2006")
		},
		"inc": func(value int) int {
			return value + 1
		},
		"dateFormat": func(time time.Time, format string) string {
			return time.Format(format)
		},
	})
	tmpl, err = tmpl.ParseFiles(
		templatesDir+"base/base.gohtml",
		templatesDir+"base/footer.gohtml",
		templatesDir+"base/navigator.gohtml",
	)
	if err != nil {
		log.Println(err.Error())
	}
	router.tmpl = tmpl

	router.Router = *mux.NewRouter()
	router.HandleFunc("/", router.Login())
	router.HandleFunc("/logout", router.Logout()).Methods(http.MethodGet)
	router.HandleFunc("/acts", router.Acts())
	router.HandleFunc("/statistics", router.Statistics())
	router.HandleFunc("/statistics/components", router.StatisticsComponents())
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, _ := router.cookie.Get(r, SessionName)
			user, ok := session.Values[userKey].(User)
			if ok {
				session.Options.MaxAge = router.sessionLifetime
				err := session.Save(r, w)
				if err != nil {
					log.Println(err.Error())
				}
				r = r.WithContext(context.WithValue(context.Background(), userKey, user))
			} else if r.URL.Path != "/" {
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	router.Use(authMiddleware)
	return router, nil
}

func (router *Router) template(templateFileName string, withBase bool, w http.ResponseWriter, data interface{}) {
	tmpl, err := router.tmpl.Clone()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
	tmpl, err = tmpl.ParseFiles(templatesDir + templateFileName)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
	if withBase {
		err = tmpl.Execute(w, data)
	} else {
		err = tmpl.ExecuteTemplate(w, templateFileName, data)
	}
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (router *Router) getUser(r *http.Request) *User {
	user := r.Context().Value(userKey)
	if user != nil {
		if user, ok := user.(User); ok {
			return &user
		}
	}
	return nil
}

func (router *Router) NewPageData(name string, footer bool, user *User) MS {
	data := MS{
		"name":      name,
		"title":     pages[name],
		"footer":    footer,
		"navigator": false,
	}
	if user != nil {
		data["user"] = user
		data["navigator"] = true
	}
	return data
}

func (router *Router) Login() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if router.getUser(r) != nil {
			http.Redirect(w, r, "acts", http.StatusFound)
		}
		page := router.NewPageData("login", false, nil)
		switch r.Method {
		case http.MethodGet:
			router.template("login.gohtml", true, w, MS{"page": page})
		case http.MethodPost:
			data, code, err := LoginProcess(w, r, router)
			if err != nil {
				log.Println(err.Error())
				http.Error(w, err.Error(), code)
				return
			}
			if data == nil {
				log.Println(err.Error())
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			if data.User == nil {
				router.template("login.fail.gohtml", true, w, MS{"page": page, "data": data})
			} else {
				http.Redirect(w, r, "acts", http.StatusFound)
			}
		}
	}
}

func (router *Router) Logout() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user := router.getUser(r)
		if user != nil {
			session, err := router.cookie.Get(r, SessionName)
			if err != nil {
				log.Println(err.Error())
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			session.Values[userKey] = nil
			err = session.Save(r, w)
			if err != nil {
				log.Println(err.Error())
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}
		http.Redirect(w, r, "", http.StatusFound)
	}
}

func (router *Router) Acts() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		page := router.NewPageData("acts", true, router.getUser(r))
		switch r.Method {
		case http.MethodGet:
			data, code, err := ActsGetPageProcess(w, r, router)
			if err != nil {
				log.Println(err.Error())
				http.Error(w, err.Error(), code)
				return
			}
			if data == nil {
				log.Println(err.Error())
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			router.template("acts.gohtml", true, w, MS{"page": page, "data": data.Bodies})
		case http.MethodPost:
			data, code, err := ActsAddProcess(w, r, router)
			if err != nil {
				log.Println(err.Error())
				http.Error(w, err.Error(), code)
				return
			}
			if data == nil {
				log.Println(err.Error())
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			router.template("acts.response.gohtml", true, w, MS{"page": page, "data": data})
		}
	}
}

func (router *Router) Statistics() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		page := router.NewPageData("statistics", true, router.getUser(r))
		data, code, err := StatisticsProcess(w, r, router)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), code)
			return
		}
		if data == nil {
			log.Println(err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		////
		router.template("statistics.gohtml", true, w, MS{"page": page, "data": data})
	}
}

func (router *Router) StatisticsComponents() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		data, code, err := StatisticsComponentsProcess(w, r, router)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), code)
			return
		}
		if data == nil {
			log.Println(err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		router.template("statistics.components.gohtml", false, w, data)
	}
}
