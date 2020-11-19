package main

import (
	"fmt"
	"strings"

	"github.com/zxsecurity/steamer/importers/util"
)

type TemplateLineParser struct{}

func (t TemplateLineParser) ParseLine(line string) ([]interface{}, error) {
	entries := make([]interface{}, 0)
	// split a line in the text file
	data := util.SplitString(line, ':', true, true)
	// check data here
	if len(data) < 4 || !strings.Contains(data[0], "@") || len(data[1]) != 40 {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	// extract the relevant data fields to form an entry
	entry := util.GenericData{
		Email:        data[0],
		Liame:        util.Reverse(data[0]),
		PasswordHash: data[1] + ":" + data[3],
		Breach:       "MyHeritage2018",
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
	importer, err := util.MakeImporter("/dumps/myheritage.txt", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
