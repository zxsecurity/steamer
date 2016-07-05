package main

import (
	"bufio"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type LinkedinData struct {
	Id           bson.ObjectId `json:"id" bson:"_id,omitempty"`
	MemberID     int           `bson:"memberid"`
	Email        string        `bson:"email"`
	Liame        string        `bson:"liame"`
	PasswordHash string        `bson:"passwordhash"`
	Password     string        `bson:"password"`
	Breach       string        `bson:"breach"`
}

func main() {
	// Connect to mongodb
	mdb, err := mgo.Dial("localhost")
	mdb.SetSocketTimeout(48 * time.Hour)
	defer mdb.Close()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to MongoDB: %v\r\n", err)
		os.Exit(1)
	}

	threads := 15
	sqlthreader := make(chan string, threads*20) // buffered to 20 * thread size
	sqldoner := make(chan bool, threads)

	for i := 0; i < threads; i++ {
		go importSQLLine(sqlthreader, mdb, sqldoner)
	}

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
		sqlthreader <- scanner.Text()
	}
	close(sqlthreader)

	// wait until all threads signal done
	for i := 0; i < threads; i++ {
		<-sqldoner
		fmt.Println("SQL Thread signaled done!")
	}
	fmt.Println("SQL imported")

	// Glob for our files
	matches, err := filepath.Glob("datafiles/*.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error globbing for files: %v\r\n", err)
		os.Exit(1)
	}

	threader := make(chan string, threads*20) // buffered to 20 * thread size
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

func importSQLLine(threader <-chan string, mgoreal *mgo.Session, doner chan<- bool) {
	// create a real collection/mgo
	mgo := mgoreal.Copy()
	c := mgo.DB("steamer").C("dumps")

	bc := 0 // insert in counts of 100

	bulk := c.Bulk()
	bulk.Unordered()

	for text := range threader {
		if bc > 1000 {
			bc = 0
			bulk.Run()
			bulk = c.Bulk()
			bulk.Unordered()
		}
		bc += 1
		// Check that these will be valid
		if (strings.Index(text, "'") + 1) >= (strings.Index(text, ",") - 1) {
			// invalid row
			fmt.Println("invalid", text)
			continue
		}
		if (strings.Index(text, ",") + 3) >= (strings.LastIndex(text, "'")) {
			// invalid row
			fmt.Println("invalid2", text)
			continue
		}
		memberidstr := text[strings.Index(text, "'")+1 : strings.Index(text, ",")-1]
		memberid, err := strconv.Atoi(memberidstr)
		if err != nil {
			fmt.Println("Error converting MemberID ", memberidstr)
			continue
		}
		email := text[strings.Index(text, ",")+3 : strings.LastIndex(text, "'")]

		entry := LinkedinData{
			MemberID: memberid,
			Email:    email,
			Liame:    Reverse(email),
			Breach:   "LinkedIn2016",
		}
		// Insert into database
		bulk.Insert(entry)
	}
	// final bulk insert
	bulk.Run()

	doner <- true
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

		// fetch a different result based on memberid vs email
		if strings.Index(data[0], "@") == -1 {
			// parse memberid if possible
			memberid, err := strconv.Atoi(data[0])
			if err != nil {
				fmt.Println("error parsing memberid", data)
				continue
			}

			err = c.Find(bson.M{"breach": "LinkedIn2016", "memberid": data[0]}).All(&results)
			if err != nil {
				fmt.Println("error finding with memberid: ", data[0], err)
				continue
			}

			if len(results) < 1 {
				// no results, a memberid we have no email for
				continue
			}

			if len(results) > 1 {
				// a complex situation, but the easiest solution is to insert a new collection that matches the previous one(s)
				entry := LinkedinData{
					MemberID:     memberid,
					Email:        results[0].Email,
					Liame:        Reverse(results[0].Email),
					Breach:       "LinkedIn2016",
					PasswordHash: data[1],
				}
				err = c.Insert(entry)
				if err != nil {
					fmt.Println("error inserting into db", err)
					continue
				}
			}

			if len(results) == 1 {
				// we can skip if the entry is already "up to date"
				if results[0].PasswordHash == data[1] {
					continue
				}

				if results[0].PasswordHash == "" {
					// Update the record with the real password hash
					c.Update(bson.M{"_id": results[0].Id}, bson.M{"$set": bson.M{"passwordhash": data[1]}})
				} else {
					// WE have a new never seen before hash for this email!
					entry := LinkedinData{
						MemberID:     memberid,
						Email:        results[0].Email,
						Liame:        Reverse(results[0].Email),
						Breach:       "LinkedIn2016",
						PasswordHash: data[1],
					}
					err = c.Insert(entry)
					if err != nil {
						fmt.Println("error inserting into db", err)
						continue
					}
				}
			}

		} else {
			// email:hash format
			err = c.Find(bson.M{"breach": "LinkedIn2016", "email": data[0]}).All(&results)

			if err != nil {
				fmt.Println("error finding with memberid: ", data[0], err)
				continue
			}

			if len(results) < 1 {
				// create a new result
				entry := LinkedinData{
					Email:        data[0],
					Liame:        Reverse(data[0]),
					Breach:       "LinkedIn2016",
					PasswordHash: data[1],
				}
				err = c.Insert(entry)
				if err != nil {
					fmt.Println("error inserting into db", err)
					continue
				}
			}

			if len(results) == 1 {
				if data[1] == results[0].PasswordHash {
					// matching hash, is fine
					continue
				}
				// update the entry
				err := c.Update(bson.M{"_id": results[0].Id}, bson.M{"$set": bson.M{"passwordhash": data[1]}})
				if err != nil {
					fmt.Println("error inserting into db", err)
					continue
				}
			}

			if len(results) > 1 {
				// complicated situation, lets just add another row with the previou results if we don't have it stored already
				skip := false
				for _, entry := range results {
					if data[1] == entry.PasswordHash {
						skip = true
					}
				}
				if skip {
					continue
				}
				// lets just add a new result I guess and hope for the best!
				entry := LinkedinData{
					MemberID:     results[0].MemberID,
					Email:        data[0],
					Liame:        Reverse(data[0]),
					Breach:       "LinkedIn2016",
					PasswordHash: data[1],
				}
				err = c.Insert(entry)
				if err != nil {
					fmt.Println("error inserting into db", err)
					continue
				}
			}
		}
	}
	doner <- true
}

func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
