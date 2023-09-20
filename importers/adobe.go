package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zxsecurity/steamer/importers/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TemplateLineParser struct{}

func (t TemplateLineParser) ParseLine(line string) ([]interface{}, error) {
	entries := make([]interface{}, 0)

	// parse out the fields we need from this horrible format
	idata := strings.Replace(line, "-", "", -1)
	// use splitN so we don't remove pipes from the hint
	data := strings.SplitN(idata, "|", 5)

	// declare nil entry
	entry := interface{}(nil)

	// check data here
	if len(data) != 5 {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	// validate/convert relevant data
	memberid, err := strconv.Atoi(data[0])
	if err != nil {
		// returns nil entry if condition fails
		entries = append(entries, entry)
		return entries, nil
	}
	hint := ""
	if len(data[4]) > 2 {
		hint = data[4][0 : len(data[4])-1]
	}

	// create our struct here
	entry = util.GenericData{
		Id:           primitive.NewObjectID(),
		MemberID:     memberid,
		Email:        data[2],
		Liame:        util.Reverse(data[2]),
		PasswordHash: data[3],
		Hint:         hint,
		Breach:       "AdobeUsers",
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
	// change filename to the default location of the dump file
	importer, err := util.MakeImporter("/dumps/adobe.txt", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
