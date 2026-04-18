package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/parse"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

//nolint:gocognit // Will be refactored in the future.
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

		torrentInfo, err := peer.GetInfo(file_bytes)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Tracker URL: " + torrentInfo.TrackerURL)
		fmt.Println("Length: " + strconv.Itoa(torrentInfo.Length))
		fmt.Println("Info Hash: " + fmt.Sprintf("%x", torrentInfo.InfoHash))
		fmt.Println("Piece Length: " + strconv.Itoa(torrentInfo.PieceLength))
		fmt.Println("Piece Hashes:")
		for i := 0; i < len(torrentInfo.PieceHashes); i++ {
			fmt.Printf("%x\n", torrentInfo.PieceHashes[i])
		}

	case "peers":
		file_name := os.Args[2]
		//nolint:gosec // This is a command-line tool. We trust the user to provide a valid file path.
		file_bytes, err := os.ReadFile(file_name)
		if err != nil {
			fmt.Println(err)
			return
		}

		torrentInfo, err := peer.GetInfo(file_bytes)
		if err != nil {
			fmt.Println(err)
			return
		}

		peers, err := peer.SendTrackerRequest(torrentInfo)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Peers:")
		for _, p := range peers {
			fmt.Println(p)
		}

	case "handshake":
		file_name := os.Args[2]
		//nolint:gosec // This is a command-line tool. We trust the user to provide a valid file path.
		file_bytes, err := os.ReadFile(file_name)
		if err != nil {
			fmt.Println(err)
			return
		}

		torrentInfo, err := peer.GetInfo(file_bytes)
		if err != nil {
			fmt.Println(err)
			return
		}

		peerIP := os.Args[3]

		peerID, err := peer.PerformHandshake(peerIP, torrentInfo)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Peer ID: %x\n", peerID)

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
