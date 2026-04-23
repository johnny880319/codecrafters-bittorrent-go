package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/bencode"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/metainfo"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/tracker"
)

func cmdDecode() {
	bencodedValue := os.Args[2]
	decoded, _, err := bencode.DecodeBencode(bencodedValue, 0)
	if err != nil {
		fmt.Println(err)
		return
	}

	jsonOutput, _ := json.Marshal(decoded)
	fmt.Println(string(jsonOutput))
}

func cmdInfo() {
	fileName := os.Args[2]
	//nolint:gosec // This is a command-line tool. We trust the user to provide a valid file path.
	fileBytes, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}

	torrentInfo, err := metainfo.Parse(fileBytes)
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
}

func cmdPeers() {
	fileName := os.Args[2]
	//nolint:gosec // This is a command-line tool. We trust the user to provide a valid file path.
	fileBytes, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}

	torrentInfo, err := metainfo.Parse(fileBytes)
	if err != nil {
		fmt.Println(err)
		return
	}

	peers, err := tracker.GetPeers(torrentInfo)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Peers:")
	for _, p := range peers {
		fmt.Println(p)
	}
}

func cmdHandshake() {
	fileName := os.Args[2]
	peerIP := os.Args[3]
	//nolint:gosec // This is a command-line tool. We trust the user to provide a valid file path.
	fileBytes, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}

	torrentInfo, err := metainfo.Parse(fileBytes)
	if err != nil {
		fmt.Println(err)
		return
	}

	conn, peerID, err := peer.Dial(peerIP, torrentInfo)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		_ = conn.Close()
	}()
	fmt.Printf("Peer ID: %x\n", peerID)
}

func cmdDownloadPiece() {
	if os.Args[2] != "-o" {
		fmt.Println("Usage: download_piece -o <output_path>")
		os.Exit(1)
	}
	outputPath := os.Args[3]
	fileName := os.Args[4]
	pieceIndex := os.Args[5]

	// Read the torrent file to get the tracker URL
	//nolint:gosec // CLI tool, file path is user-provided argument
	fileBytes, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	torrentInfo, err := metainfo.Parse(fileBytes)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Perform the tracker GET request to get a list of peers
	peers, err := tracker.GetPeers(torrentInfo)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Establish a TCP connection with a peer, and perform a handshake
	conn, _, err := peer.Dial(peers[0], torrentInfo)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	err = peer.Setup(conn)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Download the specified piece from the peer
	pieceIndexInt, err := strconv.Atoi(pieceIndex)
	if err != nil {
		fmt.Println(err)
		return
	}
	piece, err := peer.DownloadPiece(conn, torrentInfo, pieceIndexInt)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Save the downloaded piece to the output path
	//nolint:gosec // This is a command-line tool. We trust the user to provide a valid file path.
	err = os.WriteFile(outputPath, piece, 0o644)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func cmdDownload() {
	if os.Args[2] != "-o" {
		fmt.Println("Usage: download_piece -o <output_path>")
		os.Exit(1)
	}
	outputPath := os.Args[3]
	fileName := os.Args[4]

	// Read the torrent file to get the tracker URL
	//nolint:gosec // CLI tool, file path is user-provided argument
	fileBytes, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	torrentInfo, err := metainfo.Parse(fileBytes)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Perform the tracker GET request to get a list of peers
	peers, err := tracker.GetPeers(torrentInfo)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Establish a TCP connection with a peer, and perform a handshake
	conn, _, err := peer.Dial(peers[0], torrentInfo)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	err = peer.Setup(conn)
	if err != nil {
		fmt.Println(err)
		return
	}

	var pieces []byte
	for pieceIndexInt := 0; pieceIndexInt*torrentInfo.PieceLength < torrentInfo.Length; pieceIndexInt++ {
		piece, err := peer.DownloadPiece(conn, torrentInfo, pieceIndexInt)
		if err != nil {
			fmt.Println(err)
			return
		}
		pieces = append(pieces, piece...)
	}

	// Save the downloaded piece to the output path
	//nolint:gosec // This is a command-line tool. We trust the user to provide a valid file path.
	err = os.WriteFile(outputPath, pieces, 0o644)
	if err != nil {
		fmt.Println(err)
		return
	}
}
