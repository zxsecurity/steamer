package main

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/structs"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

var mdb *mgo.Session

func main() {
	var err error
	mdb, err = mgo.Dial("localhost")

	if err != nil {
		fmt.Println("Could not connect ot MongoDB: ", err)
		os.Exit(1)
	}
	defer mdb.Close()

	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/search", SearchHandler)
	r.HandleFunc("/listbreaches", ListBreaches)

	http.Handle("/", r)
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("templates/images"))))
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
	Hint         string        `bson:"hint"`
}

func SearchHandler(w http.ResponseWriter, r *http.Request) {
	// Get the searched for string
	searchterm := r.URL.Query().Get("search")
	if searchterm == "" {
		http.Error(w, "Error detecting search", http.StatusInternalServerError)
		return
	}

	breachfilter := r.URL.Query().Get("breach")
	if breachfilter == "" {
		breachfilter = "all"
	}

	// Begin a search
	mysess := mdb.Copy()
	c := mysess.DB("steamer").C("dumps")

	results := []BreachEntry{}

	var query *mgo.Query
	// TODO Remove unnessecary duplicated code here
	if breachfilter == "all" {
		query = c.Find(bson.M{"$or": []interface{}{
			bson.M{"email": bson.RegEx{fmt.Sprintf("^%v.*", regexp.QuoteMeta(searchterm)), ""}},
			bson.M{"passwordhash": searchterm},
			bson.M{"liame": bson.RegEx{fmt.Sprintf("^%v.*", regexp.QuoteMeta(Reverse(searchterm))), ""}},
		}})
	} else {
		query = c.Find(bson.M{"$and": []interface{}{
			bson.M{"breach": breachfilter},
			bson.M{"$or": []interface{}{
				bson.M{"email": bson.RegEx{fmt.Sprintf("^%v.*", regexp.QuoteMeta(searchterm)), ""}},
				bson.M{"passwordhash": searchterm},
				bson.M{"liame": bson.RegEx{fmt.Sprintf("^%v.*", regexp.QuoteMeta(Reverse(searchterm))), ""}},
			}},
		}})
	}

	// Sort if required
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "all"
	}

	if sort != "all" {
		query = query.Sort(sort)
	}

	// Get the page number
	spage := r.URL.Query().Get("page")
	// Display the first page if page is not set
	if spage == "" {
		spage = "1"
	}
	page, err := strconv.Atoi(spage)
	if err != nil {
		http.Error(w, "Error parsing limit", http.StatusInternalServerError)
		return
	}

	// Limit if required
	slimit := r.URL.Query().Get("limit")

	if slimit == "" {
		slimit = "10"
	}
	limit, err := strconv.Atoi(slimit)
	if err != nil {
		http.Error(w, "Error parsing limit", http.StatusInternalServerError)
		return
	}

	// hard limit to prevent server dying (golang will probably barf on 10k as it is since it's not iterated properly)
	if (limit > 10000) || (limit < 1) {
		limit = 10000
	}

	// Calculate limit and numner of entries to skip from page number
	skipNum := (page - 1) * limit
	// TODO: Check if skipNum will overflow
	// Check if we need to limit
	err = query.Skip(skipNum).Limit(limit).All(&results)

	if err != nil {
		fmt.Fprintf(w, "error searching %v", err)
		return
	}

	// Output based on format
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "web"
	}

	if format == "json" {
		json, err := json.Marshal(results)
		if err != nil {
			fmt.Fprintf(os.Stderr, "json encoding error: %v", err)
			http.Error(w, "Error json encoding", http.StatusInternalServerError)
			return
		}

		// replace with a bytes write rather than a string conversion
		fmt.Fprintf(w, string(json))
	} else {
		// Render the standard template with results directory
		t, err := template.ParseFiles("templates/index.html")
		if err != nil {
			fmt.Printf("error template")
		}
		// Pass in a slice map, not a map of structs
		var m []map[string]interface{}
		for _, res := range results {
			m = append(m, structs.Map(res))
		}
		templateData := struct {
			Results []map[string]interface{}
			Search  string
			Breach  string
			Sort    string
			PageNum string
			Limit   string
		}{
			m,
			searchterm,
			breachfilter,
			sort,
			spage,
			slimit,
		}
		t.Execute(w, templateData)
	}
}

// Return a JSON response of all the breaches in the database
func ListBreaches(w http.ResponseWriter, r *http.Request) {
	// db.dumps.distinct("breaches")
	mysess := mdb.Copy()
	c := mysess.DB("steamer").C("dumps")

	var results []string
	err := c.Find(nil).Distinct("breach", &results)
	if err != nil {
		fmt.Fprintf(os.Stderr, "breach search error: %v", err)
		http.Error(w, "Error searching breaches", http.StatusInternalServerError)
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

func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
