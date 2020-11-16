package main

/***
 * Format: "<email>","<password>"
 * Plain-text password
 */

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type MyFitnessPalData struct {
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
	// Get command-line flags
	verbose := flag.Bool("v", false, "Displays progress bar")
	flag.Parse()
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

	// Set filename
	filename := "myfitnesspal.txt"

	// Make ProgressBar
	var bar *pb.ProgressBar
	if *verbose {
		// Time it
		start := time.Now()
		bar = makePbar(filename)
		elapsed := time.Since(start)
		fmt.Printf("Making pbar took %s\n", elapsed)
		if bar == nil {
			fmt.Fprintf(os.Stderr, "Could not open file %v\r\n", filename)
			return
		}
	}

	for i := 0; i < threads; i++ {
		go importLine(threader, mdb, doner, bar)
	}

	// open the file
	file, err := os.Open(filename)
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
	// finish progress bar
	if bar != nil {
		bar.Finish()
	}
}

func importLine(threader <-chan string, mgoreal *mgo.Session, doner chan<- bool, bar *pb.ProgressBar) {
	// create our mongodb copy
	mgo := mgoreal.Copy()
	c := mgo.DB("steamer").C("dumps")

	bc := 0 // insert in counts of 100

	bulk := c.Bulk()
	bulk.Unordered()

	for text := range threader {
		if bc > 10000 {
			bulk.Run()
			if bar != nil {
				bar.Add(bc)
			}
			bc = 0
			bulk = c.Bulk()
			bulk.Unordered()
		}

		data := strings.SplitN(text, ",", 2)
		// Check data length
		if len(data) != 2 {
			fmt.Println("length of data should be two", data)
			if bar != nil {
				bar.Increment()
			}
			continue
		}
		// We only want users with passwords
		if data[1] == "null" {
			if bar != nil {
				bar.Increment()
			}
			continue
		}
		// Remove the quotes
		email := strings.ReplaceAll(data[0][1:len(data[0])-1], "'", "")
		password := data[1][1 : len(data[1])-1]

		// create our struct
		entry := MyFitnessPalData{
			Email:    email,
			Liame:    Reverse(email),
			Password: password,
			Breach:   "MyFitnessPal2018",
		}
		bulk.Insert(entry)
		bc += 1
	}
	// final run to be done
	bulk.Run()
	if bar != nil {
		bar.Add(bc)
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

func makePbar(filename string) *pb.ProgressBar {
	fmt.Println("Making pbar...")
	// count total number of records we need to process
	count := 0
	// open the file
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file\r\n")
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// We can add more conditions here if we want
		count += 1
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning file\r\n")
		return nil
	}

	fmt.Println("Total number of records: ", count)
	return pb.StartNew(count)
}
