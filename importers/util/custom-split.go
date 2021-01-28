package util

import (
	"strings"
	"unicode"
)

func runeUnescape(r rune) rune {
	switch r {
	case 'a':
		return '\a'
	case 'b':
		return '\b'
	case 'f':
		return '\f'
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case 'v':
		return '\v'
	case '\\':
		return '\\'
	case '\'':
		return '\''
	case '"':
		return '"'
	}
	return r
}

// SplitString splits a quoted string.
// 	query (string): the string to split
// 	sep (rune): the separator to split on
// 	keepEmpty (bool): whether to keep empty fiedls
// 	singleQuote (bool): true to use single quote, false to use double quote
func SplitString(query string, sep rune, keepEmpty bool, singleQuote bool) []string {
	components := []string{}
	unescapeNext := false
	quoted := false
	component := []rune{}
	// determine if the quotes are single or double quote
	quotes := '\''
	if !singleQuote {
		quotes = '"'
	}
	for _, r := range query {
		if quoted {
			// The previous character was a backslash, so attempt to unescape it
			// This lets us have quotes inside quotes
			if unescapeNext {
				r = runeUnescape(r)
				unescapeNext = false
				component = append(component, r)
				continue
			}
			if r == '\\' {
				unescapeNext = true
				continue
			}
			// Stop quoting the component now
			if r == quotes {
				quoted = false
			}
			component = append(component, r)
		} else {
			// We've reached the end of the component
			if (unicode.IsSpace(sep) && unicode.IsSpace(r)) || (!unicode.IsSpace(sep) && r == sep) {
				if len(component) > 0 || keepEmpty {
					components = append(components, string(component))
					component = []rune{}
				}
				continue
			}
			if r == quotes {
				quoted = true
			}
			component = append(component, r)
		}
	}
	// If there is any "incomplete" component, append it regardless
	if len(component) > 0 {
		if quoted {
			component = append(component, quotes)
		}
		components = append(components, string(component))
	}
	// Strip any quotes around quoted components
	for i, c := range components {
		if len(c) > 0 && c[0] == byte(quotes) && c[len(c)-1] == byte(quotes) {
			c = c[1 : len(c)-1]
		}
		components[i] = strings.TrimSpace(c)
	}
	return components
}
