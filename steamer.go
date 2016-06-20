package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"net/http"
	"os"
)

var mdb *mgo.Session

func main() {
	var err error
	mdb, err = mgo.Dial("localhost")
	defer mdb.Close()

	if err != nil {
		fmt.Println("Could not connect ot MongoDB: ", err)
		os.Exit(1)
	}

	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/search", SearchHandler).Methods("POST")

	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		fmt.Printf("error template")
	}
	t.Execute(w, nil)
}

type BreachEntry struct {
	Email    string
	Password string
	Hash     string
}

func SearchHandler(w http.ResponseWriter, r *http.Request) {
	c := mdb.DB("steamer").C("dumps")

	result := BreachEntry{}
	err := c.Find(bson.M{"username": "ss23"}).Limit(100).One(&result)

	if err != nil {
		fmt.Fprintf(w, "error searching %v", err)
		return
	}

	fmt.Fprintf(w, "found: %v - %v", result.Email, result.Password)
}
