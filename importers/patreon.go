package main

/***
 * This is a base/example SQL-style importer, which inserts rows into MongoDB
 * The threading is acomplished by having a reader feed lines into a function which
 *   is called in a multithreaded fashion.
 * If you require a non-threaded model, you will have to rewrite this.
 *
 * To modify this for purpose:
 * - Set a Breach (e.g. AdobeUsers, LinkedIn2016)
 * - Add any additional fields to the GenericData struct
 * - Write the importLine function as appropriate
 *
 * Ensure you set Liame correcltly (reverse of email as a string) otherwise search
 *   won't work.
 */

import (
	"bufio"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type GenericData struct {
	Id           bson.ObjectId `json:"id" bson:"_id,omitempty"`
	MemberID     int           `bson:"memberid"`
	Username     string        `bson:"username"`
	Email        string        `bson:"email"`
	Liame        string        `bson:"liame"`
	PasswordHash string        `bson:"passwordhash"`
	Password     string        `bson:"password"`
	Breach       string        `bson:"breach"`
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

	threads := 1
	threader := make(chan string, threads*20) // buffered to 20 * thread size
	doner := make(chan bool, threads)

	for i := 0; i < threads; i++ {
		go importLine(threader, mdb, doner)
	}

	// open the file
	file, err := os.Open("patreon.sql")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file\r\n")
		return
	}
	defer file.Close()

	// Quite large, perhaps not required, but better safe than sorry
	r := bufio.NewReaderSize(file, 1024*1024)

	line, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		s := string(line)

		// Lines with the INSERT line should be further parsed into individual accounts
		if strings.Contains(s, "INSERT INTO `tblUsers`") {
			// It's a valid line we should be parsing!
			threader <- s
		}

		line, isPrefix, err = r.ReadLine()
	}
	if isPrefix {
		fmt.Println("buffer size to small -- increase it in the NewReaderSize line!")
		return
	}
	if err != io.EOF {
		fmt.Fprintf(os.Stderr, "Error scanning file - %s\r\n", err)
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

		// Split out the INSERT statement so we have the tuples we need
		// Start by removing the first and last characters
		text2 := text[31 : len(text)-2]

		// Split on "),("
		users := strings.Split(text2, "),(")

		// Now loop through and process each line and INSERT as required
		for _, row := range users {
			// use splitN so we don't remove pipes from the hint
			data := strings.Split(row, ",")

			// validate/convert relevant data
			memberid, err := strconv.Atoi(data[0])
			if err != nil {
				// invalid line I guess?
				fmt.Println("Invalid data read, conversion failed", data)
				continue
			}

			// remove trailing character from hint
			if len(data) < 23 {
				// problem! error
				fmt.Println("Invalid data, epxected a len > 23, got ", len(data), row)
				continue
			}

			// create our struct
			entry := GenericData{
				MemberID:     memberid,
				Email:        data[1],
				Liame:        Reverse(data[1]),
				PasswordHash: data[4],
				Username:     data[13],
				Breach:       "Patreon",
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
