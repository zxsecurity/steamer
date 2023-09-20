package main

/***
 * 000webhost importer
 * format: colon seperated value
 * format: name:email:ip:password
 */

import (
	"fmt"

	"github.com/zxsecurity/steamer/importers/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TemplateLineParser test
type TemplateLineParser struct{}

// ParseLine parses a string and returns a list of entries to be imported
func (t TemplateLineParser) ParseLine(line string) ([]interface{}, error) {
	entries := make([]interface{}, 0)
	// ...change this to split a line in the text file
	data := util.SplitString(line, ':', true, true)
	// ...change this to check data here
	if len(data) != 4 {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	// ...change this to extract the relevant data fields to form an entry
	entry := util.GenericData{
		Id:       primitive.NewObjectID(),
		Username: data[0],
		Email:    data[1],
		Liame:    util.Reverse(data[1]),
		IP:       data[2],
		Password: data[3],
		Breach:   "000webhost",
	}
	entries = append(entries, entry)
	return entries, nil
}

// EstimateCount estimates how many entries are in a line (for the progress bar)
func (t TemplateLineParser) EstimateCount(line string) (int, error) {
	// We have one cred per line by default
	// ...change this if we have more than one cred per line
	return 1, nil
}

func main() {
	parser := TemplateLineParser{}
	// ...change filename to the default location of the dump file
	importer, err := util.MakeImporter("/dumps/000webhost.com.txt", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
