package main

import (
	"fmt"
	"strconv"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func decodeBencode(bencodedString string) (interface{}, error) {
	i := 0

	if unicode.IsDigit(rune(bencodedString[i])) {
		return decodeString(bencodedString, i)
	} else if bencodedString[i] == 'i' {
		return decodeInteger(bencodedString, i)
	} else {
		return "", fmt.Errorf("Only strings and integers are supported at the moment")
	}

}

func decodeString(bencodedString string, startIndex int) (string, error) {
	var firstColonIndex int

	for i := 0; i < len(bencodedString); i++ {
		if bencodedString[i] == ':' {
			firstColonIndex = i
			break
		}
	}

	lengthStr := bencodedString[:firstColonIndex]

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", err
	}

	return bencodedString[firstColonIndex+1 : firstColonIndex+1+length], nil
}

func decodeInteger(bencodedString string, startIndex int) (int, error) {
	var firstEIndex int

	for i := 0; i < len(bencodedString); i++ {
		if bencodedString[i] == 'e' {
			firstEIndex = i
			break
		}
	}
	return strconv.Atoi(bencodedString[startIndex+1 : firstEIndex])
}
