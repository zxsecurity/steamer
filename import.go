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
	MemberID     int
	Email        string
	PasswordHash string
	Password     string
	Breach       string
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
			// For each line of the file
			importLine(scanner.Text(), *c)
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning file %v\r\n", f)
			continue
		}
	}
}

func importSQLLine(text string, c mgo.Collection) {
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

func importLine(text string, c mgo.Collection) {
	// TODO: This check for whether email:hash or id:hash is technically not erequired every time
	// but the coding time to make ito nly happen once is bigger hit than the import delay time for checking lots

	// Split the line into x:y
	data := strings.Split(text, ":")

	// remove bogus data
	if data[0] == "null" {
		return
	}

	results := []LinkedinData{}
	var err error

	// check if it has an @ in it, and fetch record accordingly
	if strings.Index(data[0], "@") == -1 {
		// TODO: I think we can skip this and only import emails, lets do it
		return
		// ID or bogus
		// Validate that it's an ID
		i, err := strconv.Atoi(data[0])
		if (err != nil) || (i < 1) {
			// Not an email? not an ID? bogus!
			return
		}
		// Find a relevant data
		err = c.Find(bson.M{"Breach": "LinkedIn2016", "MemberID": data[0]}).All(&results)
	} else {
		// email:hash format
		err = c.Find(bson.M{"Breach": "LinkedIn2016", "Email": data[0]}).All(&results)
	}

	if err != nil {
		fmt.Println("error finding with memberid: ", data[1], err)
		return
	}
	if len(results) > 1 {
		fmt.Println("Too many potential results, please fix me!")
		return
	}
	if len(results) < 1 {
		fmt.Println("Bogus result? No pre-existing found!")
		return
	}
	// Check whether hash already exists, and if so, whether it matches
	if results[0].PasswordHash != "" {
		if results[0].PasswordHash != data[1] {
			fmt.Println("Bogus result! Hash does not match stored hash")
			return
		} else {
			// no update required
			return
		}
	}
	// Enter the hash into the database
	update := bson.M{"$set": bson.M{"PasswordHash": data[1]}}
	err = c.UpdateId(results[0].Id, update)
	if err != nil {
		fmt.Printf("Could not update record! ", err)
	}
}
