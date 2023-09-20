package main

import (
	"fmt"
	"strconv"

	"github.com/zxsecurity/steamer/importers/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TemplateLineParser struct{}

func (t TemplateLineParser) ParseLine(line string) ([]interface{}, error) {
	entries := make([]interface{}, 0)
	// split a line in the text file, use double quote instead of single quote
	data := util.SplitString(line, ',', true, false)
	// check data here
	if len(data) < 4 {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	// validate/convert relevant data
	memberid, err := strconv.Atoi(data[0])
	if err != nil || memberid == 0 {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	// extract the relevant data fields to form an entry
	entry := util.GenericData{
		Id:           primitive.NewObjectID(),
		MemberID:     memberid,
		Email:        data[3],
		Liame:        util.Reverse(data[3]),
		PasswordHash: data[2],
		Username:     data[1],
		Breach:       "StratforUsers2011",
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
	importer, err := util.MakeImporter("/dumps/stratfor_users.csv", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
