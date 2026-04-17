package peer

import (
	//nolint:gosec // BitTorrent uses SHA-1 for info hashes.
	"crypto/sha1"
	"fmt"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/parse"
)

// TorrentInfo holds the information extracted from a torrent file.
type TorrentInfo struct {
	TrackerURL  string
	Length      int
	InfoHash    []byte
	PieceLength int
	PieceHashes []string
}

// GetInfo extracts the torrent information from the given torrent file bytes.
func GetInfo(file_bytes []byte) (*TorrentInfo, error) {
	decoded, _, err := parse.DecodeBencode(string(file_bytes), 0)
	if err != nil {
		return nil, err
	}

	infoDict := decoded.(map[string]interface{})["info"].(map[string]interface{})

	trackerURL := decoded.(map[string]interface{})["announce"].(string)
	length := infoDict["length"].(int)
	infoHash, err := calculateInfoHash(file_bytes)
	if err != nil {
		return nil, err
	}
	pieceLength := infoDict["piece length"].(int)

	var pieceHashes []string
	pieceList := infoDict["pieces"].(string)

	for i := 0; i < len(pieceList); i += 20 {
		pieceHashes = append(pieceHashes, pieceList[i:i+20])
	}

	return &TorrentInfo{
		TrackerURL:  trackerURL,
		Length:      length,
		InfoHash:    infoHash,
		PieceLength: pieceLength,
		PieceHashes: pieceHashes,
	}, nil
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
