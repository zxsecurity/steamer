package main

/***
 * Importer for the Myspace 2012 breach
 * May insert two entries per user (two potential password hashes per user)
 */

import (
	"fmt"
	"strconv"

	"github.com/zxsecurity/steamer/importers/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TemplateLineParser struct{}

func (t TemplateLineParser) ParseLine(line string) ([]interface{}, error) {
	entries := make([]interface{}, 0)
	// split a line in the text file
	data := util.SplitString(line, ':', true, true)

	// check data here
	if len(data) != 5 {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	// validate/convert relevant data
	memberid, err := strconv.Atoi(data[0])
	if err != nil {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	// We may need to insert two passwords for any given user
	// Validate (and reformat) the hash(es)
	if len(data[3]) != 42 && len(data[4]) != 42 {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}

	if len(data[3]) == 42 {
		entry := util.GenericData{
			Id:           primitive.NewObjectID(),
			MemberID:     memberid,
			Email:        data[1],
			Liame:        util.Reverse(data[1]),
			Username:     data[2],
			PasswordHash: data[3][2:len(data[3])],
			Breach:       "Myspace2012",
		}
		entries = append(entries, entry)
	}

	if len(data[4]) == 42 {
		entry := util.GenericData{
			MemberID:     memberid,
			Email:        data[1],
			Liame:        util.Reverse(data[1]),
			Username:     data[2],
			PasswordHash: data[4][2:len(data[4])],
			Breach:       "Myspace2012",
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// EstimateCount estimates how many entries are in a line (for the progress bar)
func (t TemplateLineParser) EstimateCount(line string) (int, error) {
	// We have one cred per line by default
	return 1, nil
}

func main() {
	parser := TemplateLineParser{}
	// ...change filename to the default location of the dump file
	importer, err := util.MakeImporter("/dumps/Myspace.com.txt", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
