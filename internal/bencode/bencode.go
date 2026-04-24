// Package bencode provides functions to decode bencoded strings, integers, lists, and dictionaries.
package bencode

import (
	"fmt"
	"sort"
	"strconv"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

// DecodeBencode Example:
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
	case bencodedString[start] == 'd':
		return decodeDict(bencodedString, start)
	default:
		return "", start, fmt.Errorf("unsupported bencode type at position %d: %q", start, bencodedString[start])
	}
}

// EncodeBencode Example:
// - "hello" -> 5:hello
// - 12345 -> i12345e
func EncodeBencode(value interface{}) (bencodedString string, err error) {
	switch value := value.(type) {
	case string:
		return encodeString(value), nil
	case int:
		return encodeInteger(value), nil
	case []interface{}:
		return encodeList(value)
	case map[string]interface{}:
		return encodeDict(value)
	default:
		return "", fmt.Errorf("unsupported type for bencode encoding: %T", value)
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

func decodeDict(bencodedString string, start int) (decoded map[string]interface{}, end int, err error) {
	dict := make(map[string]interface{})
	i := start + 1

	for bencodedString[i] != 'e' {
		var key string
		key, i, err = decodeString(bencodedString, i)
		if err != nil {
			return nil, start, err
		}

		var value interface{}
		value, i, err = DecodeBencode(bencodedString, i)
		if err != nil {
			return nil, start, err
		}

		dict[key] = value
	}

	return dict, i + 1, nil
}

func encodeString(value string) string {
	return fmt.Sprintf("%d:%s", len(value), value)
}

func encodeInteger(value int) string {
	return fmt.Sprintf("i%de", value)
}

func encodeList(value []interface{}) (bencodedString string, err error) {
	result := "l"
	for _, element := range value {
		encodedElement, err := EncodeBencode(element)
		if err != nil {
			return "", err
		}
		result += encodedElement
	}
	result += "e"
	return result, nil
}

func encodeDict(value map[string]interface{}) (bencodedString string, err error) {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	// Sort keys to ensure deterministic encoding
	sort.Strings(keys)

	result := "d"
	for _, key := range keys {
		val := value[key]
		encodedKey := encodeString(key)
		encodedValue, err := EncodeBencode(val)
		if err != nil {
			return "", err
		}
		result += encodedKey + encodedValue
	}
	result += "e"
	return result, nil
}
