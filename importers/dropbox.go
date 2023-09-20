package main

/***
 * Dropbox bcrypt importer circa
 * No point importing the SHA's since we don't have the salt(s)
 */

import (
	"fmt"

	"github.com/zxsecurity/steamer/importers/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TemplateLineParser struct{}

func (t TemplateLineParser) ParseLine(line string) ([]interface{}, error) {
	entries := make([]interface{}, 0)
	// split a line in the text file
	data := util.SplitString(line, ':', true, true)
	// extract the relevant data fields to form an entry
	entry := util.GenericData{
		Id:           primitive.NewObjectID(),
		Email:        data[0],
		Liame:        util.Reverse(data[0]),
		PasswordHash: data[1],
		Breach:       "Dropbox2012",
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
	importer, err := util.MakeImporter("/dumps/Blowfish.txt", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
