package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

)

type LinkedinData struct {
	Id           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	MemberID     int                `bson:"memberid"`
	Email        string             `bson:"email"`
	Liame        string             `bson:"liame"`
	PasswordHash string             `bson:"passwordhash"`
	Password     string             `bson:"password"`
	Breach       string             `bson:"breach"`
}

func main() {
	// Connect to mongodb
	ctx := context.Background()
	clientOptions := options.Client().ApplyURI("mongodb://localhost").SetTimeout(48 * time.Hour)
	mdb, err := mongo.Connect(ctx, clientOptions)
	defer mdb.Disconnect(ctx)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to MongoDB: %v\r\n", err)
		os.Exit(1)
	}

	threads := 15
	sqlthreader := make(chan string, threads*20) // buffered to 20 * thread size
	sqldoner := make(chan bool, threads)

	for i := 0; i < threads; i++ {
		go importSQLLine(sqlthreader, mdb, sqldoner, ctx)
	}

	fmt.Println("Importing SQL")

	// To start with, lets parse in the MySQL dump
	sqlfile, err := os.Open("1.sql.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file %v\r\n", err)
		os.Exit(1)
	}
	defer sqlfile.Close()
	scanner := bufio.NewScanner(sqlfile)
	for scanner.Scan() {
		// For each line of the SQL
		sqlthreader <- scanner.Text()
	}
	close(sqlthreader)

	// wait until all threads signal done
	for i := 0; i < threads; i++ {
		<-sqldoner
		fmt.Println("SQL Thread signaled done!")
	}
	fmt.Println("SQL imported")

	// Glob for our files
	matches, err := filepath.Glob("datafiles/*.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error globbing for files: %v\r\n", err)
		os.Exit(1)
	}

	threader := make(chan string, threads*20) // buffered to 20 * thread size
	doner := make(chan bool, threads)

	for i := 0; i < threads; i++ {
		go importLine(threader, mdb, doner, ctx)
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

func importSQLLine(threader <-chan string, mgo *mongo.Client, doner chan<- bool, ctx context.Context) {
	// create a real collection/mgo
	c := mgo.Database("steamer").Collection("dumps")
	bc := 0 // insert in counts of 100

	var buffer []interface{}
	opts := options.InsertMany().SetOrdered(false)

	for text := range threader {
		if bc > 1000 {
			bc = 0
			c.InsertMany(ctx, buffer, opts)
			buffer = nil
		}
		bc += 1
		// Check that these will be valid
		if (strings.Index(text, "'") + 1) >= (strings.Index(text, ",") - 1) {
			// invalid row
			fmt.Println("invalid", text)
			continue
		}
		if (strings.Index(text, ",") + 3) >= (strings.LastIndex(text, "'")) {
			// invalid row
			fmt.Println("invalid2", text)
			continue
		}
		memberidstr := text[strings.Index(text, "'")+1 : strings.Index(text, ",")-1]
		memberid, err := strconv.Atoi(memberidstr)
		if err != nil {
			fmt.Println("Error converting MemberID ", memberidstr)
			continue
		}
		email := text[strings.Index(text, ",")+3 : strings.LastIndex(text, "'")]

		entry := LinkedinData{
			MemberID: memberid,
			Email:    email,
			Liame:    Reverse(email),
			Breach:   "LinkedIn2016",
		}
		// Insert into database
		buffer = append(buffer, entry)
	}
	// final bulk insert
	c.InsertMany(ctx, buffer, opts)

	doner <- true
}

func importLine(threader <-chan string, mgoreal *mongo.Client, doner chan<- bool, ctx context.Context) {

	c := mgoreal.Database("steamer").Collection("dumps")
	for text := range threader {
		// Split the line into x:y
		data := strings.Split(text, ":")

		// remove bogus data
		if data[0] == "null" {
			continue
		}
		if data[1] == "xxx" {
			continue
		}

		results := []LinkedinData{}
		var err error
		var cursor *mongo.Cursor

		// fetch a different result based on memberid vs email
		if strings.Index(data[0], "@") == -1 {
			// parse memberid if possible
			memberid, err := strconv.Atoi(data[0])
			if err != nil {
				fmt.Println("error parsing memberid", data)
				continue
			}

			cursor, err = c.Find(ctx, bson.M{"breach": "LinkedIn2016", "memberid": memberid})
			cursor.All(ctx, &results)
			if err != nil {
				fmt.Println("error finding with memberid: ", memberid, err)
				continue
			}

			if len(results) < 1 {
				// no results, a memberid we have no email for
				continue
			}

			if len(results) > 1 {
				// a complex situation, but the easiest solution is to insert a new collection that matches the previous one(s)
				entry := LinkedinData{
					Id:		   primitive.NewObjectID(),
					MemberID:     memberid,
					Email:        results[0].Email,
					Liame:        Reverse(results[0].Email),
					Breach:       "LinkedIn2016",
					PasswordHash: data[1],
				}
				_, err = c.InsertOne(ctx, entry)
				if err != nil {
					fmt.Println("error inserting into db", err)
					continue
				}
			}

			if len(results) == 1 {
				// we can skip if the entry is already "up to date"
				if results[0].PasswordHash == data[1] {
					continue
				}

				if results[0].PasswordHash == "" {
					// Update the record with the real password hash
					c.UpdateOne(ctx, bson.M{"_id": results[0].Id}, bson.M{"$set": bson.M{"passwordhash": data[1]}})
				} else {
					// WE have a new never seen before hash for this email!
					entry := LinkedinData{
						Id: 		 primitive.NewObjectID(),
						MemberID:     memberid,
						Email:        results[0].Email,
						Liame:        Reverse(results[0].Email),
						Breach:       "LinkedIn2016",
						PasswordHash: data[1],
					}
					_, err = c.InsertOne(ctx, entry)
					if err != nil {
						fmt.Println("error inserting into db", err)
						continue
					}
				}
			}

		} else {
			// email:hash format
			cursor, err = c.Find(ctx, bson.M{"breach": "LinkedIn2016", "email": data[0]})
			cursor.All(ctx, &results)

			if err != nil {
				fmt.Println("error finding with memberid: ", data[0], err)
				continue
			}

			if len(results) < 1 {
				// create a new result
				entry := LinkedinData{
					Id: 		 primitive.NewObjectID(),
					Email:        data[0],
					Liame:        Reverse(data[0]),
					Breach:       "LinkedIn2016",
					PasswordHash: data[1],
				}
				_, err = c.InsertOne(ctx, entry)
				if err != nil {
					fmt.Println("error inserting into db", err)
					continue
				}
			}

			if len(results) == 1 {
				if data[1] == results[0].PasswordHash {
					// matching hash, is fine
					continue
				}
				// update the entry
				_, err := c.UpdateOne(ctx, bson.M{"_id": results[0].Id}, bson.M{"$set": bson.M{"passwordhash": data[1]}})
				if err != nil {
					fmt.Println("error inserting into db", err)
					continue
				}
			}

			if len(results) > 1 {
				// complicated situation, lets just add another row with the previou results if we don't have it stored already
				skip := false
				for _, entry := range results {
					if data[1] == entry.PasswordHash {
						skip = true
					}
				}
				if skip {
					continue
				}
				// lets just add a new result I guess and hope for the best!
				entry := LinkedinData{
					Id: 		 primitive.NewObjectID(),
					MemberID:     results[0].MemberID,
					Email:        data[0],
					Liame:        Reverse(data[0]),
					Breach:       "LinkedIn2016",
					PasswordHash: data[1],
				}
				_, err = c.InsertOne(ctx, entry)
				if err != nil {
					fmt.Println("error inserting into db", err)
					continue
				}
			}
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
