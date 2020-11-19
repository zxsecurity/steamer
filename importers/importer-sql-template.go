package main

import (
	"fmt"

	"github.com/zxsecurity/steamer/importers/util"
)

type TemplateLineParser struct{}

func (t TemplateLineParser) ParseLine(line string) ([]interface{}, error) {
	entries := make([]interface{}, 0)
	// Parse the SQL line, ...change the target table if required
	users, err := util.ParseSQL(line, "users")
	if err != nil {
		return entries, nil
	}
	// Loop through each users
	for _, row := range users {
		// ...change this to split a line in the text file
		data := util.SplitString(row, ',', true, true)
		// declare nil entry
		entry := interface{}(nil)

		// ...check data here
		if len(data) < 23 {
			// append nil entry if condition fails
			entries = append(entries, entry)
			continue
		}

		// ...create our struct here
		entry = util.GenericData{
			MemberID:     data[0],
			Email:        data[1],
			Liame:        util.Reverse(data[1]),
			PasswordHash: data[2],
			Username:     data[3],
			Breach:       "<breach_name>",
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// EstimateCount estimates how many entries are in a line (for the progress bar)
func (t TemplateLineParser) EstimateCount(line string) (int, error) {
	count := 0
	// ...change table name if required
	users, err := util.ParseSQL(line, "users")
	if err == nil {
		count += len(users)
	}
	return count, nil
}

func main() {
	parser := TemplateLineParser{}
	// ...change filename to the default location of the dump file
	importer, err := util.MakeImporter("<filename>", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
