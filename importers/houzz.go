package main

import (
	"fmt"

	"github.com/zxsecurity/steamer/importers/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TemplateLineParser struct{}

func (t TemplateLineParser) ParseLine(line string) ([]interface{}, error) {
	entries := make([]interface{}, 0)
	// split a line in the text file
	data := util.SplitString(line, '\t', true, true)
	// check data here
	if len(data) < 5 {
		entries = append(entries, interface{}(nil))
		fmt.Println("HOUZZ ENTRIES: ", entries) //this is nil WHY
		fmt.Println("HOUZZ DATA: ", data)       //data is good, why nil after
		return entries, nil
	}
	// extract the relevant data fields to form an entry
	entry := util.GenericData{
		Id:           primitive.NewObjectID(),
		Email:        data[4],
		Liame:        util.Reverse(data[4]),
		PasswordHash: data[3],
		Name:         data[0],
		Breach:       "Houzz2019",
	}
	entries = append(entries, entry)
	fmt.Println("check in houzz parse: ", entries)
	return entries, nil
}

// EstimateCount estimates how many entries are in a line (for the progress bar)
func (t TemplateLineParser) EstimateCount(line string) (int, error) {
	return 1, nil
}

func main() {
	parser := TemplateLineParser{}
	importer, err := util.MakeImporter("/dumps/houzz.txt", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}

	importer.Run()
	importer.Finish()
}
