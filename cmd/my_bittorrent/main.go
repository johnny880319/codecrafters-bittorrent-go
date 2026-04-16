package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/parse"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

func main() {
	command := os.Args[1]

	if command == "decode" {
		bencodedValue := os.Args[2]

		decoded, _, err := parse.DecodeBencode(bencodedValue, 0)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else if command == "info" {
		file_name := os.Args[2]
		file_bytes, err := os.ReadFile(file_name)
		if err != nil {
			fmt.Println(err)
			return
		}

		decoded, _, err := parse.DecodeBencode(string(file_bytes), 0)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Tracker URL: " + decoded.(map[string]interface{})["announce"].(string))
		fmt.Println("Length: " + strconv.Itoa(decoded.(map[string]interface{})["info"].(map[string]interface{})["length"].(int)))

		infoHash, err := calculateInfoHash(file_bytes)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Info Hash: " + infoHash)

		fmt.Println("Piece Length: " + strconv.Itoa(decoded.(map[string]interface{})["info"].(map[string]interface{})["piece length"].(int)))

		fmt.Println("Piece Hashes:")
		for i := 0; i < len(decoded.(map[string]interface{})["info"].(map[string]interface{})["pieces"].(string)); i += 20 {
			fmt.Printf("%x\n", decoded.(map[string]interface{})["info"].(map[string]interface{})["pieces"].(string)[i:i+20])
		}

	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}

func calculateInfoHash(file_bytes []byte) (string, error) {
	var infoDictStart int

	for i := 0; i < len(file_bytes); i++ {
		if string(file_bytes[i:i+6]) == "4:info" {
			infoDictStart = i + 6
			break
		}
	}
	if infoDictStart == 0 {
		return "", fmt.Errorf("Info dictionary not found")
	}

	_, infoDictLen, err := parse.DecodeBencode(string(file_bytes[infoDictStart:]), 0)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return fmt.Sprintf("%x", sha1.Sum(file_bytes[infoDictStart:infoDictStart+infoDictLen])), nil
}
