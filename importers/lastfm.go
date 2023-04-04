package main

import (
	"fmt"

	"github.com/zxsecurity/steamer/importers/util"
)

type TemplateLineParser struct{}

func (t TemplateLineParser) ParseLine(line string) ([]interface{}, error) {
	entries := make([]interface{}, 0)
	data := util.SplitString(line, ' ', true, true)
	if len(data) < 4 {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	entry := util.GenericData{
		Email:        data[1],
		Liame:        util.Reverse(data[1]),
		PasswordHash: data[2],
		Username:     data[0],
		Breach:       "LastFM2012",
	}
	entries = append(entries, entry)
	return entries, nil
}

// EstimateCount estimates how many entries are in a line (for the progress bar)
func (t TemplateLineParser) EstimateCount(line string) (int, error) {
	return 1, nil
}

func main() {
	parser := TemplateLineParser{}
	importer, err := util.MakeImporter("/dumps/lastfm.txt", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
