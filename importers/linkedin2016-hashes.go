package main

import (
	"bufio"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"context"
	"os"
	"strings"
	"time"
)

type LinkedinData struct {
	Id           primitive.ObjectID `json:"id" bson:"_id,omitempty"` //need id initialized
	MemberID     int                `bson:"memberid"`
	Email        string             `bson:"email"`
	PasswordHash string             `bson:"passwordhash"`
	Password     string             `bson:"password"`
	Breach       string             `bson:"breach"`
}

func main() {
	// Connect to mongodb
	ctx := context.Background()
	clientOptions := options.Client().ApplyURI("mongodb://localhost").SetTimeout(1 * time.Hour)
	mdb, err := mongo.Connect(ctx, clientOptions)
	defer mdb.Disconnect(ctx)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to MongoDB: %v\r\n", err)
		os.Exit(1)
	}

	threads := 15
	threader := make(chan string, threads*20) // buffered to 20 * thread size
	doner := make(chan bool, threads)

	for i := 0; i < threads; i++ {
		go importLine(threader, mdb, doner, ctx)
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

func importLine(threader <-chan string, client *mongo.Client, doner chan<- bool, ctx context.Context) {

	c := client.Database("steamer").Collection("dumps")
	for text := range threader {
		// Split the line into x:y
		data := strings.SplitN(text, ":", 2)

		if len(data) != 2 {
			fmt.Println("invalid data", data)
			continue
		}

		// update any relevant results in place
		_, err := c.UpdateMany(ctx, bson.M{"breach": "LinkedIn2016", "passwordhash": data[0]},
			bson.M{"$set": bson.M{"password": data[1]}})
		if err != nil {
			fmt.Printf("error updating row %v %v %v\r\n", data[0], data[1], err)
			continue
		}
	}
	doner <- true
}
