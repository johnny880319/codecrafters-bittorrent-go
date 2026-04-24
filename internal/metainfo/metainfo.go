// Package metainfo provides functions to extract information from torrent files.
package metainfo

import (
	//nolint:gosec // BitTorrent uses SHA-1 for info hashes.
	"crypto/sha1"
	"fmt"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/bencode"
)

// MetaInfo holds the information extracted from a torrent file.
type MetaInfo struct {
	TrackerURL string
	InfoHash   []byte
	InfoDict   InfoDict
}

// InfoDict holds the length, piece length, and piece hashes extracted from the info dictionary of a torrent file.
type InfoDict struct {
	Length      int
	PieceLength int
	PieceHashes []string
}

// Parse extracts the torrent information from the given torrent file bytes.
func Parse(fileBytes []byte) (*MetaInfo, error) {
	decoded, _, err := bencode.DecodeBencode(string(fileBytes), 0)
	if err != nil {
		return nil, err
	}

	trackerURL := decoded.(map[string]interface{})["announce"].(string)
	infoHash, err := calculateInfoHash(fileBytes)
	if err != nil {
		return nil, err
	}

	infoDictRaw := decoded.(map[string]interface{})["info"].(map[string]interface{})
	infoDict, err := ParseInfoDict(infoDictRaw)
	if err != nil {
		return nil, err
	}

	return &MetaInfo{
		TrackerURL: trackerURL,
		InfoHash:   infoHash,
		InfoDict:   *infoDict,
	}, nil
}

// ParseInfoDict extracts the length, piece length, and piece hashes from the info dictionary.
func ParseInfoDict(infoDict map[string]interface{}) (*InfoDict, error) {
	length := infoDict["length"].(int)
	pieceLength := infoDict["piece length"].(int)

	var pieceHashes []string
	pieceList := infoDict["pieces"].(string)

	for i := 0; i < len(pieceList); i += 20 {
		pieceHashes = append(pieceHashes, pieceList[i:i+20])
	}

	return &InfoDict{
		Length:      length,
		PieceLength: pieceLength,
		PieceHashes: pieceHashes,
	}, nil
}

func calculateInfoHash(fileBytes []byte) ([]byte, error) {
	var infoDictStart int

	for i := 0; i < len(fileBytes); i++ {
		if string(fileBytes[i:i+6]) == "4:info" {
			infoDictStart = i + 6
			break
		}
	}
	if infoDictStart == 0 {
		return nil, fmt.Errorf("info dictionary not found")
	}

	_, infoDictLen, err := bencode.DecodeBencode(string(fileBytes[infoDictStart:]), 0)
	if err != nil {
		return nil, err
	}

	//nolint:gosec // BitTorrent info hash is defined as SHA-1.
	infoHash := sha1.Sum(fileBytes[infoDictStart : infoDictStart+infoDictLen])
	return infoHash[:], nil
}
