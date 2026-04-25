package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
		die(errors.New("usage: info <torrent_file>"))
	}
	filePath := os.Args[2]

	metaInfo, err := loadTorrent(filePath)
	die(err)

	fmt.Println("Tracker URL: " + metaInfo.TrackerURL)
	fmt.Println("Length: " + strconv.Itoa(metaInfo.InfoDict.Length))
	fmt.Println("Info Hash: " + fmt.Sprintf("%x", metaInfo.InfoHash))
	fmt.Println("Piece Length: " + strconv.Itoa(metaInfo.InfoDict.PieceLength))
	fmt.Println("Piece Hashes:")
	for i := 0; i < len(metaInfo.InfoDict.PieceHashes); i++ {
		fmt.Printf("%x\n", metaInfo.InfoDict.PieceHashes[i])
	}
}

func cmdPeers() {
	if len(os.Args) < 3 {
		die(errors.New("usage: peers <torrent_file>"))
	}
	filePath := os.Args[2]

	metaInfo, err := loadTorrent(filePath)
	die(err)

	peers, err := tracker.GetPeers(metaInfo)
	die(err)

	fmt.Println("Peers:")
	for _, p := range peers {
		fmt.Println(p)
	}
}

func cmdHandshake() {
	if len(os.Args) < 4 {
		die(errors.New("usage: handshake <torrent_file> <peer_ip>"))
	}
	filePath := os.Args[2]
	peerIP := os.Args[3]

	metaInfo, err := loadTorrent(filePath)
	die(err)

	conn, peerID, err := peer.Dial(peerIP, metaInfo)
	die(err)

	defer func() {
		err = conn.Close()
		die(err)
	}()

	fmt.Printf("Peer ID: %x\n", peerID)
}

func cmdDownloadPiece() {
	if len(os.Args) < 6 || os.Args[2] != "-o" {
		die(errors.New("usage: download_piece -o <output_path> <torrent_file> <piece_index>"))
	}
	outputPath := os.Args[3]
	filePath := os.Args[4]
	pieceIndex := os.Args[5]

	metaInfo, err := loadTorrent(filePath)
	die(err)

	peers, err := tracker.GetPeers(metaInfo)
	die(err)

	conn, _, err := peer.Dial(peers[0], metaInfo)
	die(err)

	defer func() {
		err = conn.Close()
		die(err)
	}()

	err = peer.ReadBitfield(conn)
	die(err)

	err = peer.Setup(conn)
	die(err)

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
		die(errors.New("usage: download -o <output_path> <torrent_file>"))
	}
	outputPath := os.Args[3]
	fileName := os.Args[4]

	metaInfo, err := loadTorrent(fileName)
	die(err)

	peers, err := tracker.GetPeers(metaInfo)
	die(err)

	conn, _, err := peer.Dial(peers[0], metaInfo)
	die(err)

	defer func() {
		err = conn.Close()
		die(err)
	}()

	err = peer.ReadBitfield(conn)
	die(err)

	err = peer.Setup(conn)
	die(err)

	var pieces []byte
	for pieceIndexInt := 0; pieceIndexInt*metaInfo.InfoDict.PieceLength < metaInfo.InfoDict.Length; pieceIndexInt++ {
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
		die(errors.New("usage: magnet_parse <magnet_uri>"))
	}
	magnetLink := os.Args[2]

	trackerURL, infoHash, err := magnet.Parse(magnetLink)
	die(err)

	fmt.Println("Tracker URL: " + trackerURL)
	fmt.Println("Info Hash: " + infoHash)
}

func cmdMagnetHandshake() {
	if len(os.Args) < 3 {
		die(errors.New("usage: magnet_handshake <magnet_uri>"))
	}
	magnetLink := os.Args[2]

	trackerURL, infoHash, err := magnet.Parse(magnetLink)
	die(err)

	infoHashBytes, err := hex.DecodeString(infoHash)
	die(err)

	metaInfo := &metainfo.MetaInfo{
		TrackerURL: trackerURL,
		InfoHash:   infoHashBytes,
		InfoDict: metainfo.InfoDict{
			Length: 999, // Placeholder length since we don't have the torrent file
		},
	}

	peers, err := tracker.GetPeers(metaInfo)
	die(err)

	conn, peerID, err := peer.Dial(peers[0], metaInfo)
	die(err)

	defer func() {
		err = conn.Close()
		die(err)
	}()

	extensionID, err := peer.ExtensionHandshake(conn)
	die(err)

	fmt.Printf("Peer ID: %x\n", peerID)
	fmt.Printf("Peer Metadata Extension ID: %d\n", extensionID)
}

func cmdMagnetInfo() {
	if len(os.Args) < 3 {
		die(errors.New("usage: magnet_handshake <magnet_uri>"))
	}
	magnetLink := os.Args[2]

	trackerURL, infoHash, err := magnet.Parse(magnetLink)
	die(err)

	infoHashBytes, err := hex.DecodeString(infoHash)
	die(err)

	metaInfo := &metainfo.MetaInfo{
		TrackerURL: trackerURL,
		InfoHash:   infoHashBytes,
		InfoDict: metainfo.InfoDict{
			Length: 999, // Placeholder length since we don't have the torrent file
		},
	}

	peers, err := tracker.GetPeers(metaInfo)
	die(err)

	conn, _, err := peer.Dial(peers[0], metaInfo)
	die(err)

	defer func() {
		err = conn.Close()
		die(err)
	}()

	extensionID, err := peer.ExtensionHandshake(conn)
	die(err)

	infoDict, err := peer.ExtensionMetadata(conn, extensionID)
	die(err)

	fmt.Println("Tracker URL: " + trackerURL)
	fmt.Println("Length: " + strconv.Itoa(infoDict.Length))
	fmt.Println("Info Hash: " + fmt.Sprintf("%x", infoHashBytes))
	fmt.Println("Piece Length: " + strconv.Itoa(infoDict.PieceLength))
	fmt.Println("Piece Hashes:")
	for i := 0; i < len(infoDict.PieceHashes); i++ {
		fmt.Printf("%x\n", infoDict.PieceHashes[i])
	}
}

func cmdMagnetDownloadPiece() {
	if len(os.Args) < 6 || os.Args[2] != "-o" {
		die(errors.New("usage: magnet_download_piece -o <output_path> <magnet_uri> <piece_index>"))
	}
	outputPath := os.Args[3]
	magnetLink := os.Args[4]
	pieceIndex := os.Args[5]

	trackerURL, infoHash, err := magnet.Parse(magnetLink)
	die(err)

	infoHashBytes, err := hex.DecodeString(infoHash)
	die(err)

	metaInfo := &metainfo.MetaInfo{
		TrackerURL: trackerURL,
		InfoHash:   infoHashBytes,
		InfoDict: metainfo.InfoDict{
			Length: 999, // Placeholder length since we don't have the torrent file
		},
	}

	peers, err := tracker.GetPeers(metaInfo)
	die(err)

	conn, _, err := peer.Dial(peers[0], metaInfo)
	die(err)

	defer func() {
		err = conn.Close()
		die(err)
	}()

	extensionID, err := peer.ExtensionHandshake(conn)
	die(err)

	infoDict, err := peer.ExtensionMetadata(conn, extensionID)
	die(err)

	metaInfo.InfoDict = *infoDict

	err = peer.Setup(conn)
	die(err)

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

func cmdMagnetDownload() {
	if len(os.Args) < 5 || os.Args[2] != "-o" {
		die(errors.New("usage: magnet_download -o <output_path> <magnet_uri>"))
	}
	outputPath := os.Args[3]
	magnetLink := os.Args[4]

	trackerURL, infoHash, err := magnet.Parse(magnetLink)
	die(err)

	infoHashBytes, err := hex.DecodeString(infoHash)
	die(err)

	metaInfo := &metainfo.MetaInfo{
		TrackerURL: trackerURL,
		InfoHash:   infoHashBytes,
		InfoDict: metainfo.InfoDict{
			Length: 999, // Placeholder length since we don't have the torrent file
		},
	}

	peers, err := tracker.GetPeers(metaInfo)
	die(err)

	conn, _, err := peer.Dial(peers[0], metaInfo)
	die(err)

	defer func() {
		err = conn.Close()
		die(err)
	}()

	extensionID, err := peer.ExtensionHandshake(conn)
	die(err)

	infoDict, err := peer.ExtensionMetadata(conn, extensionID)
	die(err)

	metaInfo.InfoDict = *infoDict

	err = peer.Setup(conn)
	die(err)

	var pieces []byte
	for pieceIndexInt := 0; pieceIndexInt*metaInfo.InfoDict.PieceLength < metaInfo.InfoDict.Length; pieceIndexInt++ {
		piece, err := peer.DownloadPiece(conn, metaInfo, pieceIndexInt)
		die(err)

		pieces = append(pieces, piece...)
	}

	// Save the downloaded piece to the output path
	//nolint:gosec // This is a command-line tool. We trust the user to provide a valid file path.
	err = os.WriteFile(outputPath, pieces, 0o644)
	die(err)
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

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
