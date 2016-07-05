package main

import (
	"bufio"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"os"
	"strings"
	"time"
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

	// set timeout high to prevent timeouts (weird!)
	mdb.SetSocketTimeout(1 * time.Hour)

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
	file, err := os.Open("siph0n_LinkedIn_46M_cracked.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file\r\n")
		return
	}
	defer file.Close()

	// TODO: Threaded!
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
		// Split the line into x:y
		data := strings.SplitN(text, ":", 2)

		if len(data) != 2 {
			fmt.Println("invalid data", data)
			continue
		}

		// update any relevant results in place
		_, err := c.UpdateAll(bson.M{"breach": "LinkedIn2016", "passwordhash": data[0]},
			bson.M{"$set": bson.M{"password": data[1]}})
		if err != nil {
			fmt.Printf("error updating row %v %v %v\r\n", data[0], data[1], err)
			continue
		}
	}
	doner <- true
}
