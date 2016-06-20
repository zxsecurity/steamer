package main

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"os"
	"path/filepath"
	"fmt"
	"bufio"
	"strings"
	"strconv"
)

type LinkedinData struct {
	Id           bson.ObjectId `json:"id" bson:"_id,omitempty"`
	MemberID     int           `bson:"memberid"`
	Email        string        `bson:"email"`
	PasswordHash string        `bson:"passwordhash"`
	Password     string        `bson:"password"`
	Breach       string        `bson:"breach"`
}

func main() {
	// Connect to mongodb
	mdb, err := mgo.Dial("localhost")
	defer mdb.Close()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to MongoDB: %v\r\n", err)
		os.Exit(1)
	}

	// Our collection for generic dump data
	c := mdb.DB("steamer").C("dumps")

	fmt.Println("Importing SQL")

	// To start with, lets parse in the MySQL dump
	sqlfile, err := os.Open("1.sql.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file %v\r\n", err)
		os.Exit(1)
	}
	defer sqlfile.Close()
	scanner := bufio.NewScanner(sqlfile)
	for scanner.Scan() {
		// For each line of the SQL
		importSQLLine(scanner.Text(), *c)
	}

	fmt.Println("SQL imported")

	// Glob for our files
	matches, err := filepath.Glob("datafiles/*.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error globbing for files: %v\r\n", err)
		os.Exit(1)
	}

	threads := 15
	threader := make(chan string, threads * 20) // buffered to 20 * thread size
	doner := make(chan bool, threads)

	for i := 0; i < threads; i++ {
		go importLine(threader, mdb, doner)
	}

	for _, f := range matches {
		fmt.Printf("Parsing file %v...\r\n", f)

		// open the file
		file, err := os.Open(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file %v\r\n", f)
			continue
		}

		defer file.Close()

		// TODO: Threaded!
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			// For each line of the file write to the channel
			threader <- scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning file %v\r\n", f)
			continue
		}
	}
	// close the threader channel
	close(threader)

	// wait until all threads signal done
	for i := 0; i < threads; i++ {
		<-doner
		fmt.Println("Thread signaled done!")
	}
}

func importSQLLine(text string, c mgo.Collection) {
	// Check that these will be valid
	if (strings.Index(text, "'") + 1) <= (strings.Index(text, ",") - 1) {
		// invalid row
		return
	}
	if (strings.Index(text, ",") + 3) <= (strings.LastIndex(text, "'")) {
		// invalid row
		return
	}
	memberidstr := text[strings.Index(text, "'") + 1:strings.Index(text, ",") - 1]
	memberid, err := strconv.Atoi(memberidstr)
	if err != nil {
		fmt.Println("Error converting MemberID ", memberidstr)
		return
	}
	email := text[strings.Index(text, ",") + 3:strings.LastIndex(text, "'")]

	entry := LinkedinData{
		MemberID: memberid,
		Email: email,
		Breach: "LinkedIn2016",
	}
	// Insert into database
	c.Insert(entry)
}

func importLine(threader <-chan string, mgoreal *mgo.Session, doner chan<- bool) {
	// create our mongodb copy
	mgo := mgoreal.Copy()

	c := mgo.DB("steamer").C("dumps")
	for text := range threader {
		// Split the line into x:y
		data := strings.Split(text, ":")

		// remove bogus data
		if data[0] == "null" {
			continue
		}
		if data[1] == "xxx" {
			continue
		}

		results := []LinkedinData{}
		var err error

		// Ignore memberid:hash entries, only parse emails
		if strings.Index(data[0], "@") == -1 {
			continue
		}

		// email:hash format
		err = c.Find(bson.M{"breach": "LinkedIn2016", "email": data[0], "passwordhash": data[1]}).All(&results)

		if err != nil {
			fmt.Println("error finding with memberid: ", data[1], err)
			continue
		}

		if len(results) > 1 {
			fmt.Println("Too many potential results, please fix me!")
			continue
		}

		// Only create a new result if this is a new password hash combination
		if len(results) == 1 {
			continue
		}

		entry := LinkedinData{
			Email: data[0],
			Breach: "LinkedIn2016",
			PasswordHash: data[1],
		}
		err = c.Insert(entry)
		if err != nil {
			fmt.Println("error inserting into db", err)
			continue
		}
	}
	doner <- true
}
