// Package util provides common code for all the importers
package util

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cheggaaa/pb/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"context"
)

type GenericData struct {
	Id           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	MemberID     int           `bson:"memberid"`
	Email        string        `bson:"email"`
	Liame        string        `bson:"liame"`
	PasswordHash string        `bson:"passwordhash"`
	Password     string        `bson:"password"`
	Username     string        `bson:"username"`
	Hint         string        `bson:"hint"`
	Breach       string        `bson:"breach"`
	IP           string        `bson:"ip"`
	Name         string        `bson:"name"`
}

type LineParser interface {
	ParseLine(line string) ([]interface{}, error)
	EstimateCount(line string) (int, error)
}

type Importer struct {
	parser     LineParser
	bar        *pb.ProgressBar
	numThreads int
	threader   chan string
	doner      chan bool
	client      *mongo.Client
	verbose    bool
	fileName   string
}

func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// MakeImporter constructs and returns an importer
func MakeImporter(defaultFileName string, parser LineParser, numThreads int) (*Importer, error) {

	// Get command-line flags
	verbose := flag.Bool("v", false, "Displays progress bar")
	fileName := *flag.String("i", defaultFileName, "Path to dump file")
	flag.Parse()

	// Connect to mongodb
	ctx := context.Background()
	clientOptions := options.Client().ApplyURI("mongodb://localhost").SetTimeout(24 * time.Hour)
	mdb, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("Could not connect to MongoDB: %v\r\n", err)
	}
	// don't defer mdb.Disconnect(ctx) here, it causes importing to not work properly

	// Make ProgressBar
	var bar *pb.ProgressBar
	if *verbose {
		// Time it
		start := time.Now()
		bar, err = makePbar(fileName, parser)
		elapsed := time.Since(start)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Making pbar took %s\n", elapsed)
	}
	importer := Importer{
		parser:     parser,
		bar:        bar,
		numThreads: numThreads,
		threader:   make(chan string, numThreads*20),
		doner:      make(chan bool, numThreads),
		mongo:      mdb,
		verbose:    *verbose,
		fileName:   fileName,
	}
	return &importer, nil
}

// makePbar estimates the number of entries in a dump
// and returns a progress bar
func makePbar(fileName string, parser LineParser) (*pb.ProgressBar, error) {
	fmt.Println("Making pbar...")
	// count total number of records we need to process
	count := 0
	// open the file
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("Could not open file %v\r\n", fileName)
	}
	defer file.Close()

	// Quite large, perhaps not required, but better safe than sorry
	r := bufio.NewReaderSize(file, 1024*1024)

	line, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		s := string(line)
		num, err_parser := parser.EstimateCount(s)
		if err_parser == nil {
			count += num
		}
		line, isPrefix, err = r.ReadLine()
	}
	if isPrefix {
		return nil, fmt.Errorf("buffer size to small -- increase it in the NewReaderSize line!")
	}
	if err != io.EOF {
		return nil, fmt.Errorf("Error scanning file - %s\r\n", err)
	}

	fmt.Println("Total number of records: ", count)
	return pb.StartNew(count), nil
}

func (i *Importer) Run() {
	// this part should have the threader <- r.ReadLine() loop and the <- doner loop
	for ind := 0; ind < i.numThreads; ind++ {
		go i.importLine()
	}

	// open the file
	file, err := os.Open(i.fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v (path: %s, file: %s)\n", err, func() string { dir, _ := os.Getwd(); return dir }(), i.fileName)
		return
	}
	defer file.Close()

	// Quite large, perhaps not required, but better safe than sorry
	r := bufio.NewReaderSize(file, 1024*1024)

	line, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		s := string(line)
		i.threader <- s

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
	close(i.threader)

	// wait until all threads signal done
	for ind := 0; ind < i.numThreads; ind++ {
		<-i.doner
		fmt.Println("Thread signaled done!")
	}

}

// Finish finishes the importing process
func (i *Importer) Finish() {
	i.client.Disconnect(context.Background())
	// finish progress bar
	if i.bar != nil {
		i.bar.Finish()
	}
}

// importLine calls the parser's ParseLine function to parse a line
// and inserts it to mongodb
func (i *Importer) importLine() {
	// create our mongodb copy
	mgo := i.client
	ctx := context.Background()
	c := mgo.Database("steamer").Collection("dumps")

	bc := 0
	var buffer []interface{}
	opts := options.InsertMany().SetOrdered(false)

	for text := range i.threader {
		if bc > 10000 {
			c.InsertMany(ctx, buffer, opts)
			if i.bar != nil {
				i.bar.Add(bc)
			}
			bc = 0
			buffer = nil
		}
		entries, err := i.parser.ParseLine(text)
		
		if err != nil {
			fmt.Println(err)
			continue
		}
		// Insert every entry
		for _, entry := range entries {
			if entry == nil {
				if i.bar != nil {
					i.bar.Increment()
				}
			} else {
				buffer = append(buffer, entry)
				bc += 1
			}
		}
	}
	// final run to be done
	c.InsertMany(ctx, buffer, opts)
	if i.bar != nil {
		i.bar.Add(bc)
	}
	i.doner <- true
}
