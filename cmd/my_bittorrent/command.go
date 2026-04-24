package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/bencode"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/magnet"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/metainfo"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/tracker"
)

func cmdDecode() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: decode <bencoded_value>")
		os.Exit(1)
	}
	bencodedValue := os.Args[2]
	decoded, _, err := bencode.DecodeBencode(bencodedValue, 0)
	die(err)

	jsonOutput, _ := json.Marshal(decoded)
	fmt.Println(string(jsonOutput))
}

func cmdInfo() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: info <torrent_file>")
		os.Exit(1)
	}
	filePath := os.Args[2]
	metaInfo, err := loadTorrent(filePath)
	die(err)

	fmt.Println("Tracker URL: " + metaInfo.TrackerURL)
	fmt.Println("Length: " + strconv.Itoa(metaInfo.Length))
	fmt.Println("Info Hash: " + fmt.Sprintf("%x", metaInfo.InfoHash))
	fmt.Println("Piece Length: " + strconv.Itoa(metaInfo.PieceLength))
	fmt.Println("Piece Hashes:")
	for i := 0; i < len(metaInfo.PieceHashes); i++ {
		fmt.Printf("%x\n", metaInfo.PieceHashes[i])
	}
}

func cmdPeers() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: peers <torrent_file>")
		os.Exit(1)
	}
	filePath := os.Args[2]
	metaInfo, err := loadTorrent(filePath)
	die(err)

	peers, err := tracker.GetPeers(metaInfo)
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
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: handshake <torrent_file> <peer_ip>")
		os.Exit(1)
	}
	filePath := os.Args[2]
	peerIP := os.Args[3]
	metaInfo, err := loadTorrent(filePath)
	die(err)

	conn, peerID, err := peer.Dial(peerIP, metaInfo)
	die(err)

	defer func() {
		_ = conn.Close()
	}()
	fmt.Printf("Peer ID: %x\n", peerID)
}

func cmdDownloadPiece() {
	if len(os.Args) < 6 || os.Args[2] != "-o" {
		fmt.Fprintln(os.Stderr, "Usage: download_piece -o <output_path> <torrent_file> <piece_index>")
		os.Exit(1)
	}
	outputPath := os.Args[3]
	filePath := os.Args[4]
	pieceIndex := os.Args[5]

	metaInfo, err := loadTorrent(filePath)
	die(err)

	conn, err := connectPeer(metaInfo)
	die(err)

	defer func() {
		_ = conn.Close()
	}()

	// Download the specified piece from the peer
	pieceIndexInt, err := strconv.Atoi(pieceIndex)
	die(err)

	piece, err := peer.DownloadPiece(conn, metaInfo, pieceIndexInt)
	die(err)

	// Save the downloaded piece to the output path
	//nolint:gosec // This is a command-line tool. We trust the user to provide a valid file path.
	err = os.WriteFile(outputPath, piece, 0o644)
	die(err)
}

func cmdDownload() {
	if len(os.Args) < 5 || os.Args[2] != "-o" {
		fmt.Fprintln(os.Stderr, "Usage: download -o <output_path> <torrent_file>")
		os.Exit(1)
	}
	outputPath := os.Args[3]
	fileName := os.Args[4]

	metaInfo, err := loadTorrent(fileName)
	die(err)

	conn, err := connectPeer(metaInfo)
	die(err)
	defer func() {
		_ = conn.Close()
	}()

	var pieces []byte
	for pieceIndexInt := 0; pieceIndexInt*metaInfo.PieceLength < metaInfo.Length; pieceIndexInt++ {
		piece, err := peer.DownloadPiece(conn, metaInfo, pieceIndexInt)
		die(err)
		pieces = append(pieces, piece...)
	}

	// Save the downloaded piece to the output path
	//nolint:gosec // This is a command-line tool. We trust the user to provide a valid file path.
	err = os.WriteFile(outputPath, pieces, 0o644)
	die(err)
}

func cmdMagnetParse() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: magnet_parse <magnet_uri>")
		os.Exit(1)
	}
	magnetLink := os.Args[2]
	trackerURL, infoHash, err := magnet.Parse(magnetLink)
	die(err)

	fmt.Println("Tracker URL: " + trackerURL)
	fmt.Println("Info Hash: " + infoHash)
}

func cmdMagnetHandshake() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: magnet_handshake <magnet_uri>")
		os.Exit(1)
	}
	magnetLink := os.Args[2]
	trackerURL, infoHash, err := magnet.Parse(magnetLink)
	die(err)

	infoHashBytes, err := hex.DecodeString(infoHash)
	die(err)

	metaInfo := &metainfo.MetaInfo{
		TrackerURL: trackerURL,
		InfoHash:   infoHashBytes,
		Length:     999, // Placeholder length since we don't have the torrent file
	}

	peers, err := tracker.GetPeers(metaInfo)
	die(err)

	conn, peerID, err := peer.Dial(peers[0], metaInfo)
	die(err)
	defer func() {
		_ = conn.Close()
	}()

	err = peer.ExtensionHandshake(conn)
	die(err)

	fmt.Printf("Peer ID: %x\n", peerID)
}

func loadTorrent(path string) (*metainfo.MetaInfo, error) {
	//nolint:gosec // This is a command-line tool. We trust the user to provide a valid file path.
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	torrentInfo, err := metainfo.Parse(fileBytes)
	if err != nil {
		return nil, err
	}
	return torrentInfo, nil
}

func connectPeer(metaInfo *metainfo.MetaInfo) (net.Conn, error) {
	peers, err := tracker.GetPeers(metaInfo)
	if err != nil {
		return nil, err
	}

	conn, _, err := peer.Dial(peers[0], metaInfo)
	if err != nil {
		return nil, err
	}

	err = peer.Setup(conn)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
