package main

import (
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/dchest/uniuri"
	"github.com/ewhal/pygments"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

const (
	ADDRESS = "http://localhost:9900"
	LENGTH  = 6
	TEXT    = "$ <command> | curl -F 'p=<-' " + ADDRESS + "\n"
	PORT    = ":9900"
)

type Response struct {
	ID     string `json:"id"`
	HASH   string `json:"hash"`
	URL    string `json:"url"`
	SIZE   int    `json:"size"`
	DELKEY string `json:"delkey"`
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func generateName() string {
	s := uniuri.NewLen(LENGTH)
	db, err := sql.Open("sqlite3", "./database.db")
	check(err)

	query, err := db.Query("select id from pastebin")
	for query.Next() {
		var id string
		err := query.Scan(&id)
		if err != nil {

		}
		if id == s {
			generateName()
		}
	}
	db.Close()

	return s

}
func hash(paste []byte) string {
	hasher := sha1.New()
	hasher.Write(paste)
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	return sha
}

func save(raw []byte) []string {
	p := raw[86 : len(raw)-46]

	db, err := sql.Open("sqlite3", "./database.db")
	check(err)

	sha := hash(p)
	id := generateName()
	url := ADDRESS + "/p/" + id
	delKey := uniuri.NewLen(40)
	paste := html.EscapeString(string(p))

	stmt, err := db.Prepare("INSERT INTO pastebin(id, hash, data, delkey) values(?,?,?,?)")
	check(err)
	_, err = stmt.Exec(id, sha, paste, delKey)
	check(err)
	db.Close()
	return []string{id, sha, url, paste, delKey}

}

func delHandler(w http.ResponseWriter, r *http.Request) {
}
func saveHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	output := vars["output"]
	switch r.Method {
	case "POST":
		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}

		values := save(buf)
		b := &Response{
			ID:     values[0],
			HASH:   values[1],
			URL:    values[2],
			SIZE:   len(values[3]),
			DELKEY: values[4],
		}

		switch output {
		case "json":

			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(b)

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case "xml":
			x, err := xml.MarshalIndent(b, "", "  ")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/xml")
			w.Write(x)

		default:
			io.WriteString(w, values[2]+"\n")
		}
	}

}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, TEXT)
}
func langHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	paste := vars["pasteId"]
	lang := vars["lang"]
	s := getPaste(paste)
	highlight := pygments.Highlight(html.UnescapeString(s), lang, "html", "full, style=autumn,linenos=True, lineanchors=True,anchorlinenos=True,", "utf-8")
	io.WriteString(w, highlight)

}

func getPaste(paste string) string {
	param1 := html.EscapeString(paste)
	db, err := sql.Open("sqlite3", "./database.db")
	var s string
	err = db.QueryRow("select data from pastebin where id=?", param1).Scan(&s)
	db.Close()
	check(err)

	if err == sql.ErrNoRows {
		return "Error invalid paste"
	} else {
		return html.UnescapeString(s)
	}

}
func pasteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paste := vars["pasteId"]
	s := getPaste(paste)
	io.WriteString(w, s)

}

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", rootHandler)
	router.HandleFunc("/p/{pasteId}", pasteHandler)
	router.HandleFunc("/p/{pasteId}/{lang}", langHandler)
	router.HandleFunc("/save", saveHandler)
	router.HandleFunc("/save/{output}", saveHandler)
	router.HandleFunc("/del/{pasteId}/{delKey}", delHandler)
	err := http.ListenAndServe(PORT, router)
	if err != nil {
		log.Fatal(err)
	}

}
