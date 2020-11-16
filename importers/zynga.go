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
 * - Write the importLine function as appropriate (this expects INSERT (),(),() style data)
 *
 * Ensure you set Liame correcltly (reverse of email as a string) otherwise search
 *   won't work.
 */

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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
	filename := "Zynga.com.sql"

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

	// Quite large, perhaps not required, but better safe than sorry
	r := bufio.NewReaderSize(file, 1024*1024)

	line, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		s := string(line)

		// Lines with the INSERT line should be further parsed into individual accounts
		if strings.Contains(s, "INSERT INTO `users`") {
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
		// Split out the INSERT statement so we have the tuples we need
		// Start by removing the first and last characters
		text2 := text[31 : len(text)-2]

		// Split on "),("
		users := strings.Split(text2, "),(")

		// Now loop through and process each line and INSERT as required
		tmpBulkCount := 0
		for _, row := range users {
			// use splitN so we don't remove pipes from the hint
			data := strings.Split(row, ",")

			// validate/convert relevant data
			memberid, err := strconv.Atoi(data[0])
			if err != nil {
				fmt.Println("Invalid data read, conversion failed", data)
				continue
			}

			// remove trailing character from hint
			if len(data) < 23 {
				fmt.Println("Invalid data, epxected a len > 23, got ", len(data), row)
				continue
			}

			// ensure email has an '@' in it
			if !strings.Contains(data[1], "@") {
				continue
			}
			// Check if our user is deleted
			emailStr := strings.Split(data[1], "-")

			if len(emailStr) == 3 && emailStr[1] == "deleted" {
				// Deleted user
				continue
			} else {
				// crypted_udid_password may just be the device ID so we ignore it
				// we only include crypted_custom_password which may be the actual user password
				// Import the user
				if len(data[9]) == 42 && len(data[7]) == 42 {
					// remove space in email
					email := strings.ReplaceAll(data[1][1:len(data[1])-1], " ", "")
					// remove some trailing \' and \" in emails
					if strings.Contains(email, "\\") {
						email = strings.Split(email, "\\")[0]
					}
					username := data[2][1 : len(data[2])-1]
					crypted_custom_password := data[9][1 : len(data[9])-1]
					salt := data[7][1 : len(data[7])-1]
					pwdHashSalt := salt + ":" + crypted_custom_password
					// create our struct
					entry := GenericData{
						MemberID:     memberid,
						Email:        email,
						Liame:        Reverse(email),
						PasswordHash: pwdHashSalt,
						Username:     username,
						Breach:       "Zynga2019",
					}
					bulk.Insert(entry)
					tmpBulkCount += 1
				}
			}
		}
		bc += tmpBulkCount
		if bar != nil {
			bar.Add(len(users) - tmpBulkCount)
		}
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

	// Quite large, perhaps not required, but better safe than sorry
	r := bufio.NewReaderSize(file, 1024*1024)

	line, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		text := string(line)

		// Lines with the INSERT line should be further parsed into individual accounts
		if strings.Contains(text, "INSERT INTO `users`") {
			text2 := text[31 : len(text)-2]
			count += len(strings.Split(text2, "),("))
		}

		line, isPrefix, err = r.ReadLine()
	}
	if isPrefix {
		fmt.Println("buffer size to small -- increase it in the NewReaderSize line!")
		return nil
	}
	if err != io.EOF {
		fmt.Fprintf(os.Stderr, "Error scanning file - %s\r\n", err)
		return nil
	}
	fmt.Println("Total number of records: ", count)
	return pb.StartNew(count)
}
