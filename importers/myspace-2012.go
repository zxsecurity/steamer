package main

/***
 * Importer for the Myspace 2012 breach
 * May insert two entries per user (two potential password hashes per user)
 */

import (
	"bufio"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"os"
	"strconv"
	"strings"
	"time"
)

type MyspaceData struct {
	Id           bson.ObjectId `json:"id" bson:"_id,omitempty"`
	MemberID     int           `bson:"memberid"`
	Email        string        `bson:"email"`
	Liame        string        `bson:"liame"`
	PasswordHash string        `bson:"passwordhash"`
	Password     string        `bson:"password"`
	Breach       string        `bson:"breach"`
	Name         string        `bson:"name"`
}

func main() {
	// Connect to mongodb
	mdb, err := mgo.Dial("localhost")
	defer mdb.Close()

	mdb.SetSocketTimeout(24 * time.Hour)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to MongoDB: %v\r\n", err)
		os.Exit(1)
	}

	threads := 15
	threader := make(chan string, threads*20) // buffered to 20 * thread size
	doner := make(chan bool, threads)

	for i := 0; i < threads; i++ {
		go importLine(threader, mdb, doner)
	}

	// open the file
	file, err := os.Open("Myspace.com.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file\r\n")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// For each line of the file write to the channel
		threader <- scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning file\r\n")
		return
	}

	// close the threader channel
	close(threader)

	// wait until all threads signal done
	for i := 0; i < threads; i++ {
		<-doner
		fmt.Println("Thread signaled done!")
	}
}

func importLine(threader <-chan string, mgoreal *mgo.Session, doner chan<- bool) {
	// create our mongodb copy
	mgo := mgoreal.Copy()
	c := mgo.DB("steamer").C("dumps")

	bc := 0 // insert in counts of 100

	bulk := c.Bulk()
	bulk.Unordered()

	for text := range threader {
		if bc > 10000 {
			bc = 0
			bulk.Run()
			bulk = c.Bulk()
			bulk.Unordered()
		}
		bc += 1

		// use splitN so we don't remove pipes from the hint
		data := strings.Split(text, ":")

		if len(data) != 5 {
			fmt.Println("invalid data read", len(data), data)
			continue
		}

		// validate/convert relevant data
		memberid, err := strconv.Atoi(data[0])
		if err != nil {
			// invalid line I guess?
			fmt.Println("Invalid data read, conversion failed", data)
			continue
		}

		// We may need to insert two passwords for any given user
		// Validate (and reformat) the hash(es)
		if len(data[3]) == 42 {
			// create our struct
			entry := MyspaceData{
				MemberID:     memberid,
				Email:        data[1],
				Liame:        Reverse(data[1]),
				Name:         data[2],
				PasswordHash: data[3][2:len(data[3])],
				Breach:       "Myspace2012",
			}
			bulk.Insert(entry)
		}

		if len(data[4]) == 42 {
			// create our struct
			entry := MyspaceData{
				MemberID:     memberid,
				Email:        data[1],
				Liame:        Reverse(data[1]),
				Name:         data[2],
				PasswordHash: data[4][2:len(data[4])],
				Breach:       "Myspace2012",
			}
			bulk.Insert(entry)
		}

	}
	// final run to be done
	bulk.Run()
	doner <- true
}

func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
