package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zxsecurity/steamer/importers/util"
)

type TemplateLineParser struct{}

func (t TemplateLineParser) ParseLine(line string) ([]interface{}, error) {
	entries := make([]interface{}, 0)
	// Parse the SQL line
	users, err := util.ParseSQL(line, "tblUsers")
	if err != nil {
		return entries, nil
	}
	// Loop through each users
	for _, row := range users {
		// split on comma, use single quote
		data := util.SplitString(row, ',', true, true)
		// declare nil entry
		entry := interface{}(nil)

		// check data here
		if len(data) < 23 {
			entries = append(entries, entry)
			continue
		}
		// validate/convert relevant data
		memberid, err := strconv.Atoi(data[0])
		if err != nil || !strings.Contains(data[1], "@") {
			// append nil entry if condition fails
			entries = append(entries, entry)
			continue
		}

		// create our struct here
		entry = util.GenericData{
			MemberID:     memberid,
			Email:        data[1],
			Liame:        util.Reverse(data[1]),
			PasswordHash: data[4],
			Username:     data[13],
			Breach:       "Patreon2015",
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// EstimateCount estimates how many entries are in a line (for the progress bar)
func (t TemplateLineParser) EstimateCount(line string) (int, error) {
	count := 0
	users, err := util.ParseSQL(line, "tblUsers")
	if err == nil {
		count += len(users)
	}
	return count, nil
}

func main() {
	parser := TemplateLineParser{}
	importer, err := util.MakeImporter("/dumps/patreon.sql", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
