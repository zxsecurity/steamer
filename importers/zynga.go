package main

/***
 * Format: SQL
 * Hashing Method: sha1(<salt>, $pwd)
 * Hash format to crack: <salt>:<crypted_custom_password>
 */

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
	users, err := util.ParseSQL(line, "users")
	if err != nil {
		return entries, nil
	}
	// Loop through each users
	for _, row := range users {
		data := util.SplitString(row, ',', true, true)
		// declare nil entry
		entry := interface{}(nil)

		// check the data length
		if len(data) < 23 {
			fmt.Println("Invalid data, expected a len >= 23, got ", len(data), data)
			fmt.Println(row)
			entries = append(entries, entry)
			continue
		}
		// memeberid
		memberid, err := strconv.Atoi(data[0])
		if err != nil {
			fmt.Println("Invalid data read, conversion failed", data)
			entries = append(entries, entry)
			continue
		}
		// ensure email has an '@' in it
		if !strings.Contains(data[1], "@") {
			entries = append(entries, entry)
			continue
		}
		// Check if our user is deleted
		emailStr := strings.Split(data[1], "-")
		if len(emailStr) == 3 && emailStr[1] == "deleted" {
			entries = append(entries, entry)
			continue
		}
		// Ensure hash length
		if len(data[9]) == 40 && len(data[7]) == 40 {
			// remove space in email
			email := strings.ReplaceAll(data[1], " ", "")
			// remove some trailing \' and \" in emails
			if strings.Contains(email, "\\") {
				email = strings.Split(email, "\\")[0]
			}
			username := data[2]
			pwdHashSalt := data[7] + ":" + data[9]
			// create our struct
			entry = util.GenericData{
				MemberID:     memberid,
				Email:        email,
				Liame:        util.Reverse(email),
				PasswordHash: pwdHashSalt,
				Username:     username,
				Breach:       "Zynga2019",
			}
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// EstimateCount estimates how many entries are in a line (for the progress bar)
func (t TemplateLineParser) EstimateCount(line string) (int, error) {
	count := 0
	users, err := util.ParseSQL(line, "users")
	if err == nil {
		count += len(users)
	}
	return count, nil
}

func main() {
	parser := TemplateLineParser{}
	importer, err := util.MakeImporter("/dumps/Zynga.com.sql", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
