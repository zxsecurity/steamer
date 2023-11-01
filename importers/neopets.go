package main

/***
 * Neopets
 * WARNING: Be sure to combine all the files into one:
 * `cat neopets_2013_68M/*.txt > neopets_combined.txt`
 */

import (
	"fmt"

	"github.com/zxsecurity/steamer/importers/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"gopkg.in/mgo.v2/bson"
)

type NeopetsData struct {
	Id        bson.ObjectId `json:"id" bson:"_id,omitempty"`
	Username  string        `bson:"username"`
	Password  string        `bson:"password"`
	Email     string        `bson:"email"`
	Liame     string        `bson:"liame"`
	DOB       string        `bson:"dob"`
	Country   string        `bson:"country"`
	Gender    string        `bson:"gender"`
	IPAddress string        `bson:"ipaddress"`
	Name      string        `bson:"name"`
	Breach    string        `bson:"breach"`
}
type TemplateLineParser struct{}

func (t TemplateLineParser) ParseLine(line string) ([]interface{}, error) {
	entries := make([]interface{}, 0)
	// split a line in the text file
	data := util.SplitString(line, ':', true, true)
	if len(data) < 8 {
		entries = append(entries, interface{}(nil))
		return entries, nil
	}
	// extract the relevant data fields to form an entry
	entry := NeopetsData{
		Id:        primitive.NewObjectID(),
		Username:  data[0],
		Password:  data[1],
		Email:     data[2],
		Liame:     util.Reverse(data[2]),
		DOB:       data[3],
		Country:   data[4],
		Gender:    data[5],
		IPAddress: data[6],
		Name:      data[7],
		Breach:    "Neopets2013",
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
	importer, err := util.MakeImporter("/dumps/neopets_combined.txt", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
