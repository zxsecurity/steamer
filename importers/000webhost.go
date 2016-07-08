package main

/***
 * 000webhost importer
 * format: colon seperated value
 * format: name:email:ip:password
 */
import (
	"bufio"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"os"
	"strings"
	"time"
)

type GenericData struct {
	Id       bson.ObjectId `json:"id" bson:"_id,omitempty"`
	Username string        `bson:"username"`
	Email    string        `bson:"email"`
	Liame    string        `bson:"liame"`
	Password string        `bson:"password"`
	IP       string        `bson:"IP"`
	Breach   string        `bson:"breach"`
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
	file, err := os.Open("000webhost.com.txt")
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

		// SplitN in case password has a : in it
		data := strings.SplitN(text, ":", 4)

		if len(data) != 4 {
			fmt.Println("invalid data: ", text)
			continue
		}

		// create our struct
		entry := GenericData{
			Username: data[0],
			Email:    data[1],
			Liame:    Reverse(data[1]),
			IP:       data[2],
			Password: data[3],
			Breach:   "000webhost",
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
