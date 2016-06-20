package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"net/http"
	"os"
	"time"
	"encoding/json"
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
	Id           bson.ObjectId `json:"id" bson:"_id,omitempty"`
	MemberID     int           `bson:"memberid"`
	Email        string        `bson:"email"`
	PasswordHash string        `bson:"passwordhash"`
	Password     string        `bson:"password"`
	Breach       string        `bson:"breach"`
}

func SearchHandler(w http.ResponseWriter, r *http.Request) {
	// Get the searched for string
	err := r.ParseForm()
	if err != nil {
		fmt.Println(fmt.Sprintf("[%v] Could parse form data - %v", time.Now(), err))
		http.Error(w, "Error parsing form", http.StatusInternalServerError)
		return
	}

	searchterm := ""
	if val, ok := r.Form["search"]; ok {
		if val[0] != "" {
			searchterm = val[0]
		}
	}

	if searchterm == "" {
		http.Error(w, "Error detecting search", http.StatusInternalServerError)
		return
	}

	// Begin a search
	c := mdb.DB("steamer").C("dumps")

	results := []BreachEntry{}
	err = c.Find(bson.M{"$or": []interface{}{
		bson.M{"email": bson.RegEx{fmt.Sprintf(".*%v.*", searchterm), ""}},
		bson.M{"passwordhash": bson.RegEx{fmt.Sprintf(".*%v.*", searchterm), ""}},
	}}).Limit(100).All(&results)

	if err != nil {
		fmt.Fprintf(w, "error searching %v", err)
		return
	}

	json, err := json.Marshal(results)
	if err != nil {
		fmt.Fprintf(os.Stderr, "json encoding error: %v", err)
		http.Error(w, "Error json encoding", http.StatusInternalServerError)
		return
	}

	// replace with a bytes write rather than a string conversion
	fmt.Fprintf(w, string(json))
}
