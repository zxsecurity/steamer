package main

/***
 * Format: "<email>","<password>"
 * Plain-text password
 */

import (
	"fmt"
	"strings"

	"github.com/zxsecurity/steamer/importers/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TemplateLineParser struct{}

func (t TemplateLineParser) ParseLine(line string) ([]interface{}, error) {
	// TODO: Handle some errors
	entries := make([]interface{}, 0)
	// strip single quotes
	line = strings.ReplaceAll(line, "'", "")
	// code to parse a line into its data blobs
	data := util.SplitString(line, ',', true, false)
	// check the data
	if len(data) != 2 || data[1] == "null" || !strings.Contains(data[0], "@") {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	// create our struct
	entry := util.GenericData{
		Id:       primitive.NewObjectID(),
		Email:    data[0],
		Liame:    util.Reverse(data[0]),
		Password: data[1],
		Breach:   "MyFitnessPal2018",
	}
	entries = append(entries, entry)
	return entries, nil
}

// EstimateCount estimates how many entries are in a line (for the progress bar)
func (t TemplateLineParser) EstimateCount(line string) (int, error) {
	// We have one cred per line by default
	return 1, nil
}

func main() {
	parser := TemplateLineParser{}
	importer, err := util.MakeImporter("/dumps/myfitnesspal.txt", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
