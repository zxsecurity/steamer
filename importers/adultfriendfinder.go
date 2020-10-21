package main

/***
 * Format: SQL
 * Hashing Method: hashcat mode 100 -> SHA1, sha1($pwd, 'blib')
 * Hash format to crack: <password_hash>
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

type AdultFriendFinderData struct {
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
	filename := "ffadult_edb_users.sql"

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
			strippedRow := strings.ReplaceAll(row, "'", "")
			data := strings.Split(strippedRow, ",")

			// format: pwsid, handle, email, lastvisit, password, send_email, show_lang.....
			if len(data) < 4 {
				// problem! error
				fmt.Println("Invalid data, epxected a len > 4, got ", len(data), strippedRow)
				continue
			}

			// We remove the underscore and convert it to member ID
			memberid, err := strconv.Atoi(strings.Replace(data[0], "_", "", 1))
			if err != nil {
				fmt.Println("Invalid data read, conversion failed", data)
				continue
			}

			// There may be multiple emails in the email column, get the index of the last email
			// The default index of email is 2
			lastEmailIndex := 2
			for i := 2; i < len(data); i++ {
				// extremely rough check if it's a valid email
				if strings.Contains(data[i], "@") {
					lastEmailIndex = i
				} else {
					break
				}
			}
			// We only care about users with password hash
			if len(data[lastEmailIndex+2]) == 40 {
				// Import all the emails
				for i := 2; i <= lastEmailIndex; i++ {
					// Parse relevant entries
					entry := AdultFriendFinderData{
						MemberID:     memberid,
						Email:        data[i],
						Liame:        Reverse(data[i]),
						PasswordHash: data[4],
						Username:     data[1],
						Breach:       "AdultFriendFinder2016",
					}
					bulk.Insert(entry)
					tmpBulkCount += 1
				}
			} else {
				// Here we insert those emails with commas
				if len(data) > 4 {
					hasPwd := false
					for i := 4; i < len(data); i++ {
						if len(data[i]) == 40 {
							hasPwd = true
						}
					}
					// Check if we skipped over some password hash
					// This may happen if we have comma in the email
					if hasPwd {
						tmpData := strings.Split(row, "','")
						// insert this entry
						entry := AdultFriendFinderData{
							MemberID:     memberid,
							Email:        tmpData[2],
							Liame:        Reverse(tmpData[2]),
							PasswordHash: tmpData[4],
							Username:     tmpData[1],
							Breach:       "AdultFriendFinder2016",
						}
						bulk.Insert(entry)
						tmpBulkCount += 1
					}
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
