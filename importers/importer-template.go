package main

/***
 * This is a base/example CSV-style importer, which inserts rows into MongoDB
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
	"os"
	"strconv"
	"strings"
	"time"
)

type GenericData struct {
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
	file, err := os.Open("cred")
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

		// parse out the fields we need from this horrible format
		idata := strings.Replace(text, "-", "", -1)

		// use splitN so we don't remove pipes from the hint
		data := strings.SplitN(idata, "|", 5)

		// validate/convert relevant data
		memberid, err := strconv.Atoi(data[0])
		if err != nil {
			// invalid line I guess?
			fmt.Println("Invalid data read, conversion failed", data)
			continue
		}

		// remove trailing character from hint
		if len(data) != 5 {
			// problem! error
			fmt.Println("Invalid data, lots failed", data)
			continue
		}

		hint := ""
		if len(data[4]) > 2 {
			hint = data[4][0 : len(data[4])-1]
		}

		// create our struct
		entry := AdobeData{
			MemberID:     memberid,
			Email:        data[2],
			Liame:        Reverse(data[2]),
			PasswordHash: data[3],
			Hint:         hint,
			Breach:       false(), // Set the Breach here
		}
		bulk.Insert(entry)
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
