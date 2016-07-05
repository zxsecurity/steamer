package main

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

type StratforData struct {
	Id           bson.ObjectId `json:"id" bson:"_id,omitempty"`
	MemberID     int           `bson:"memberid"`
	Email        string        `bson:"email"`
	Liame        string        `bson:"liame"`
	PasswordHash string        `bson:"passwordhash"`
	Password     string        `bson:"password"`
	Name         string        `bson:"name"`
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
	file, err := os.Open("stratfor_users.csv")
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
	for text := range threader {
		// parse out the fields we need from this horrible format
		idata := strings.Replace(text, "\"", "", -1)

		data := strings.Split(idata, ",")

		// validate/convert relevant data
		memberid, err := strconv.Atoi(data[0])
		if err != nil {
			// invalid line I guess?
			fmt.Println("Invalid data read, conversion failed", data)
			continue
		}

		if memberid == 0 {
			fmt.Println("invalid memberid", data)
			continue
		}

		if len(data) < 4 {
			fmt.Println("invalid data", data)
			continue
		}

		// create our struct
		entry := StratforData{
			MemberID:     memberid,
			Email:        data[3],
			Liame:        Reverse(data[3]),
			PasswordHash: data[2],
			Name:         data[1],
			Breach:       "StratforUsers",
		}
		err = c.Insert(entry)
		if err != nil {
			fmt.Println("error inserting ", data, err)
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
