package parse

import (
	"fmt"
	"strconv"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func DecodeBencode(bencodedString string, start int) (decoded interface{}, end int, err error) {
	switch {
	case unicode.IsDigit(rune(bencodedString[start])):
		return decodeString(bencodedString, start)
	case bencodedString[start] == 'i':
		return decodeInteger(bencodedString, start)
	case bencodedString[start] == 'l':
		return decodeList(bencodedString, start)
	default:
		return "", start, fmt.Errorf("Only strings and integers are supported at the moment")
	}
}

func decodeString(bencodedString string, start int) (decoded string, end int, err error) {
	var firstColonIndex int

	for i := start; i < len(bencodedString); i++ {
		if bencodedString[i] == ':' {
			firstColonIndex = i
			break
		}
	}

	lengthStr := bencodedString[start:firstColonIndex]

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", start, err
	}

	return bencodedString[firstColonIndex+1 : firstColonIndex+1+length], firstColonIndex + 1 + length, nil
}

func decodeInteger(bencodedString string, start int) (decoded int, end int, err error) {
	var firstEIndex int

	for i := start; i < len(bencodedString); i++ {
		if bencodedString[i] == 'e' {
			firstEIndex = i
			break
		}
	}
	value, err := strconv.Atoi(bencodedString[start+1 : firstEIndex])
	if err != nil {
		return 0, start, err
	}
	return value, firstEIndex + 1, nil
}

func decodeList(bencodedString string, start int) (decoded []interface{}, end int, err error) {
	list := make([]interface{}, 0)
	i := start + 1

	for bencodedString[i] != 'e' {
		var element interface{}
		element, i, err = DecodeBencode(bencodedString, i)
		if err != nil {
			return nil, start, err
		}
		list = append(list, element)
	}

	return list, i + 1, nil
}
