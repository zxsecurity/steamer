package main

/***
 * Format: PostgreSQL
 * Hashing Method: hashcat mode 3200 -> bcrypt
 * Hash format to crack:
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
	data := util.SplitString(line, '\t', true, false)
	// validate length
	if len(data) < 30 || len(data) < 1 {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	// validate/convert relevant data
	memberid, err := strconv.Atoi(data[0])
	if err != nil {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	// check if we have pwd hash
	pwdHash := data[8]
	if len(pwdHash) <= 2 {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	// extract the relevant data fields to form an entry
	entry := util.GenericData{
		Id:           primitive.NewObjectID(),
		MemberID:     memberid,
		Email:        data[5],
		Liame:        util.Reverse(data[5]),
		PasswordHash: pwdHash,
		Name:         data[6] + " " + data[7],
		Breach:       "NitroPDF2020",
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
	importer, err := util.MakeImporter("/dumps/nitrocloud.tsv", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
