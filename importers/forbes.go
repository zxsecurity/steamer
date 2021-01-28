package main

import (
	"fmt"
	"strconv"

	"github.com/zxsecurity/steamer/importers/util"
)

type TemplateLineParser struct{}

func (t TemplateLineParser) ParseLine(line string) ([]interface{}, error) {
	entries := make([]interface{}, 0)
	// split a line in the text file
	data := util.SplitString(line, ';', true, false)
	// validate/convert relevant data
	memberid, err := strconv.Atoi(data[0])
	if err != nil {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	if memberid == 0 || len(data) < 10 {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}

	// extract the relevant data fields to form an entry
	entry := util.GenericData{
		MemberID:     memberid,
		Email:        data[4],
		Liame:        util.Reverse(data[4]),
		PasswordHash: data[2],
		Name:         data[9],
		Username:     data[1],
		Breach:       "ForbesUsers",
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
	importer, err := util.MakeImporter("/dumps/forbes_wp_users.txt", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
