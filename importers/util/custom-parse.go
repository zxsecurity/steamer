package util

import (
	"fmt"
	"strings"
)

// ParseSQL parses a line of the SQL statement by extracting
// the relevant INSERT line into the target table
func ParseSQL(line string, table string) ([]string, error) {
	// Make sure we have an INSERT line for the table we target
	keyword := "INSERT INTO `" + table + "`"
	if !strings.Contains(line, keyword) {
		return nil, fmt.Errorf("Not an insert line\r\n")
	}
	dataLine := line[31 : len(line)-2]
	// For SQL files, we have multiple entries in one line
	users := strings.Split(dataLine, "),(")
	return users, nil
}
