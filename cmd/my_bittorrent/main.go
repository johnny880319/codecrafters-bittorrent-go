package main

import (
	//nolint:gosec // BitTorrent uses SHA-1 for info hashes.
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/parse"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

func main() {
	command := os.Args[1]

	switch command {
	case "decode":

		bencodedValue := os.Args[2]

		decoded, _, err := parse.DecodeBencode(bencodedValue, 0)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))

	case "info":
		file_name := os.Args[2]
		//nolint:gosec // This is a command-line tool. We trust the user to provide a valid file path.
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
		length := decoded.(map[string]interface{})["info"].(map[string]interface{})["length"].(int)
		fmt.Println("Length: " + strconv.Itoa(length))

		infoHash, err := calculateInfoHash(file_bytes)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Info Hash: " + fmt.Sprintf("%x", infoHash))

		pieceLength := decoded.(map[string]interface{})["info"].(map[string]interface{})["piece length"].(int)
		fmt.Println("Piece Length: " + strconv.Itoa(pieceLength))

		fmt.Println("Piece Hashes:")
		for i := 0; i < len(decoded.(map[string]interface{})["info"].(map[string]interface{})["pieces"].(string)); i += 20 {
			fmt.Printf("%x\n", decoded.(map[string]interface{})["info"].(map[string]interface{})["pieces"].(string)[i:i+20])
		}

	case "peers":
		file_name := os.Args[2]
		//nolint:gosec // This is a command-line tool. We trust the user to provide a valid file path.
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

		trackerURL := decoded.(map[string]interface{})["announce"].(string)

		infoHash, err := calculateInfoHash(file_bytes)
		if err != nil {
			fmt.Println(err)
			return
		}
		pieceLength := decoded.(map[string]interface{})["info"].(map[string]interface{})["piece length"].(int)

		err = peer.SendTrackerRequest(trackerURL, infoHash, pieceLength)
		if err != nil {
			fmt.Println(err)
			return
		}

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}

func calculateInfoHash(file_bytes []byte) ([]byte, error) {
	var infoDictStart int

	for i := 0; i < len(file_bytes); i++ {
		if string(file_bytes[i:i+6]) == "4:info" {
			infoDictStart = i + 6
			break
		}
	}
	if infoDictStart == 0 {
		return nil, fmt.Errorf("info dictionary not found")
	}

	_, infoDictLen, err := parse.DecodeBencode(string(file_bytes[infoDictStart:]), 0)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	//nolint:gosec // BitTorrent info hash is defined as SHA-1.
	infoHash := sha1.Sum(file_bytes[infoDictStart : infoDictStart+infoDictLen])
	return infoHash[:], nil
}
