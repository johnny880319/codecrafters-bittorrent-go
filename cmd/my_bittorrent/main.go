package main

import (
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
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
