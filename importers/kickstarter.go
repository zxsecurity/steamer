package main

/***
 * Format: <email>:<password_hash>:<salt>
 * Hashing Method: hashcat mode 19500 -> Ruby on Rails Restful-Authentication
 * Hash format to crack: <password_hash>:<salt>:f5945d1c74d3502f8a3de8562e5bf21fe3fec887
 */

import (
	"fmt"
	"strings"

	"github.com/zxsecurity/steamer/importers/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TemplateLineParser struct{}

func (t TemplateLineParser) ParseLine(line string) ([]interface{}, error) {
	// TODO: Handle some errors
	entries := make([]interface{}, 0)
	// code to parse a line into its data blobs
	data := util.SplitString(line, ':', true, true)
	// data := strings.Split(line, ":")
	// check the data
	if len(data) < 3 || data[1] == "NULL" || strings.HasPrefix(data[0], " ") || !strings.Contains(data[0], "@") {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	// create our struct
	pwdHash := data[1] + ":" + data[2] + ":f5945d1c74d3502f8a3de8562e5bf21fe3fec887"
	entry := util.GenericData{
		Id:           primitive.NewObjectID(),
		Email:        data[0],
		Liame:        util.Reverse(data[0]),
		PasswordHash: pwdHash,
		Breach:       "KickStarter2014",
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
	importer, err := util.MakeImporter("/dumps/kickstarter.txt", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
