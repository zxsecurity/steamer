package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type NeopetsData struct {
	Id        bson.ObjectId `json:"id" bson:"_id,omitempty"`
	Username  string        `bson:"username"`
	Password  string        `bson:"password"`
	Email     string        `bson:"email"`
	Liame     string        `bson:"liame"`
	DOB       string        `bson:"dob"`
	Country   string        `bson:"country"`
	Gender    string        `bson:"gender"`
	IPAddress string        `bson:"ipaddress"`
	Name      string        `bson:"name"`
	Breach    string        `bson:"breach"`
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

	// Glob for our files
	matches, err := filepath.Glob("*.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error globbing for files: %v\r\n", err)
		os.Exit(1)
	}

	threads := 15
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

		data := strings.Split(text, ":")

		// create our struct
		entry := NeopetsData{
			Username:  data[0],
			Password:  data[1],
			Email:     data[2],
			Liame:     Reverse(data[2]),
			DOB:       data[3],
			Country:   data[4],
			Gender:    data[5],
			IPAddress: data[6],
			Name:      data[7],
			Breach:    "Neopets",
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
