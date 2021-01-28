package main

/***
 * Format: SQL
 * Hashing Method: hashcat mode 100 -> SHA1, sha1($pwd, 'blib')
 * Hash format to crack: <password_hash>:blib
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
		// Make sure the initial single quote is there
		if !strings.HasPrefix(row, "'") {
			row = "'" + row
		}
		data := util.SplitString(row, ',', true, true)
		// declare nil entry
		entry := interface{}(nil)

		// check the data length
		if len(data) < 5 {
			fmt.Println("Invalid data, expected a len >= 5, got ", len(data), data)
			entries = append(entries, entry)
			continue
		}
		// memeberid
		memberid, err := strconv.Atoi(strings.Replace(data[0], "_", "", 1))
		if err != nil {
			fmt.Println("Invalid data read, conversion failed", data)
			entries = append(entries, entry)
			continue
		}
		// hash length
		if len(data[4]) != 40 {
			// fmt.Println("No hash...skipping", data)
			entries = append(entries, entry)
			continue
		}

		// We first insert those with only one email
		// we can't split on commas arbitrarily becuz of edge cases like abc,@test.com
		if strings.Count(data[2], "@") == 1 {
			data[2] = strings.ReplaceAll(data[2], ",", "")
			// strip any remaining quotes
			data[2] = strings.ReplaceAll(data[2], "'", "")
			entry = util.GenericData{
				MemberID:     memberid,
				Email:        data[2],
				Liame:        util.Reverse(data[2]),
				PasswordHash: data[4],
				Username:     data[1],
				Breach:       "AdultFriendFinder2016",
			}
			entries = append(entries, entry)
			continue
		}
		// this is the case of multiple emails
		emails := util.SplitString(data[2], ',', true, true)

		for i := 0; i < len(emails); i++ {
			// extremely rough check if it's a valid email
			if strings.Contains(emails[i], "@") {
				// strip any remaining quotes
				emails[i] = strings.ReplaceAll(emails[i], "'", "")
				entry = util.GenericData{
					MemberID:     memberid,
					Email:        emails[i],
					Liame:        util.Reverse(emails[i]),
					PasswordHash: data[4],
					Username:     data[1],
					Breach:       "AdultFriendFinder2016",
				}
				entries = append(entries, entry)
			} else {
				entry := interface{}(nil)
				entries = append(entries, entry)
			}
		}
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
	importer, err := util.MakeImporter("/dumps/ffadult_edb_users.sql", parser, 15)
	if err != nil {
		fmt.Println(err)
		return
	}
	importer.Run()
	importer.Finish()
}
