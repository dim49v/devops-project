package main

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"
)

func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "..")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

func isTitleElement(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "title"
}

func traverse(n *html.Node) (string, bool) {
	if isTitleElement(n) {
		return n.FirstChild.Data, true
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result, ok := traverse(c)
		if ok {
			return result, ok
		}
	}

	return "", false
}

func GetHtmlTitle(body []byte) (string, bool) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		panic("Fail to parse html")
	}
	data, ok := traverse(doc)
	return strings.TrimSpace(data), ok
}
func NewJar() *cookiejar.Jar {
	jar, _ := cookiejar.New(nil)
	return jar
}

var (
	client = &http.Client{nil, nil, NewJar(), time.Second}
)

type Case struct {
	Method      string // GET по-умолчанию в http.NewRequest если передали пустую строку
	Path        string
	Query       string
	Status      int
	Result      string
	Title       string
	ContentType string
	Boundary    string
}

const (
	Api               = "/"
	ApiLogout         = "/logout"
	ApiActs           = "/acts"
	ApiStatistics     = "/statistics"
	ApiStatisticsComp = "/statistics/components"
)

// CaseResponse
type CR map[string]interface{}

func TestMyApi(t *testing.T) {
	gob.Register(User{})
	gob.Register(userKey)
	host := os.Getenv("MYSQL_HOST")
	port := os.Getenv("MYSQL_PORT")
	user := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")
	database := os.Getenv("MYSQL_DATABASE")

	DSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true", user, password, host, port, database)
	var db *sql.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("mysql", DSN)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		panic(err)
	}
	router, _ := NewRouter(db)
	ts := httptest.NewServer(router)

	cases := []Case{
		Case{
			Path:   ApiActs,
			Method: http.MethodGet,
			Query:  "",
			Status: http.StatusOK,
			Title:  "Вход",
		},
		Case{
			Path:   Api,
			Method: http.MethodGet,
			Query:  "",
			Status: http.StatusOK,
			Title:  "Вход",
		},
		Case{
			Path:   Api,
			Method: http.MethodPost,
			Query:  "login=test&password=test",
			Status: http.StatusOK,
			Title:  "Вход",
		},
		Case{
			Path:   Api,
			Method: http.MethodPost,
			Query:  "login=&password=test",
			Status: http.StatusOK,
			Title:  "Вход",
		},
		Case{
			Path:   Api,
			Method: http.MethodPost,
			Query:  "login=admin&password=admin",
			Status: http.StatusOK,
			Title:  "Акты",
		},
		Case{
			Path:   Api,
			Method: http.MethodPost,
			Query:  "login=admin&password=admin",
			Status: http.StatusOK,
			Title:  "Акты",
		},
		Case{
			Path:   ApiActs,
			Method: http.MethodGet,
			Query:  "",
			Status: http.StatusOK,
			Title:  "Акты",
		},
		Case{
			Path:   ApiActs,
			Method: http.MethodPost,
			Query:  "body_sel=1&body_part_sel=6&Select27=343&Select28=126&Select29=151&actNumber=123123&date=2022-04-01&additional=1345",
			Status: http.StatusInternalServerError,
		},
		Case{
			Path:        ApiActs,
			Method:      http.MethodPost,
			Query:       "body_sel=1&body_part_sel=6&Select27=343&Select28=126&Select29=151&actNumber=123123&date=2022-04-01&additional=1345",
			ContentType: "multipart/form-data;",
			Status:      http.StatusOK,
			Title:       "Акты",
		},
		Case{
			Path:   ApiStatistics,
			Method: http.MethodGet,
			Query:  "",
			Status: http.StatusOK,
			Title:  "Статистика",
		},
		Case{
			Path:   ApiStatisticsComp,
			Method: http.MethodGet,
			Query:  "year=2022",
			Status: http.StatusInternalServerError,
		},
		Case{
			Path:   ApiStatisticsComp,
			Method: http.MethodGet,
			Query:  "bodyPart=0",
			Status: http.StatusInternalServerError,
		},
		Case{
			Path:   ApiStatisticsComp,
			Method: http.MethodGet,
			Query:  "year=2022&body_part=0",
			Status: http.StatusOK,
		},
		Case{
			Path:   ApiLogout,
			Method: http.MethodGet,
			Query:  "",
			Status: http.StatusOK,
		},
		Case{
			Path:   ApiActs,
			Method: http.MethodGet,
			Query:  "",
			Status: http.StatusOK,
			Title:  "Вход",
		},
	}

	runTests(t, ts, cases)
	clearDB(db)
}

func clearDB(db *sql.DB) {
	db.Exec("DELETE FROM act_component WHERE act_id = (SELECT id FROM act WHERE number = '123123')")
	db.Exec("DELETE FROM act WHERE number = '123123'")
}
func runTests(t *testing.T, ts *httptest.Server, cases []Case) {
	for idx, item := range cases {
		var (
			err error
			req *http.Request
		)

		caseName := fmt.Sprintf("case %d: [%s] %s %s", idx, item.Method, item.Path, item.Query)

		if item.Method == http.MethodPost {
			if item.ContentType == "" {
				reqBody := strings.NewReader(item.Query)
				req, err = http.NewRequest(item.Method, ts.URL+item.Path, reqBody)
				req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			} else {
				var b bytes.Buffer
				w := multipart.NewWriter(&b)
				values := strings.Split(item.Query, "&")
				for _, query := range values {
					key := strings.Split(query, "=")[0]
					r := strings.NewReader(strings.Split(query, "=")[1])
					var fw io.Writer
					if fw, err = w.CreateFormField(key); err != nil {
						continue
					}
					if _, err = io.Copy(fw, r); err != nil {
						continue
					}
				}
				w.Close()
				req, err = http.NewRequest(item.Method, ts.URL+item.Path, &b)
				req.Header.Add("Content-Type", w.FormDataContentType())

			}
		} else {
			req, err = http.NewRequest(item.Method, ts.URL+item.Path+"?"+item.Query, nil)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("[%s] request error: %v", caseName, err)
			continue
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		if resp.StatusCode != item.Status {
			t.Errorf("[%s] expected http status %v, got %v", caseName, item.Status, resp.StatusCode)
			continue
		}

		if item.Title != "" {
			title, _ := GetHtmlTitle(body)
			if title != item.Title {
				t.Errorf("[%s] HTML titles not equal: expected %s, got %s", caseName, item.Title, title)
				continue
			}
		}
		if string(item.Result) != "" && item.Result != string(body) {
			t.Errorf("[%s] bodies not equal", caseName)
			continue
		}
		t.Logf("[%s] request success", caseName)
	}
}
