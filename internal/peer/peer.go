// Package peer implements the BitTorrent peer wire protocol.
package peer

import (
	"bytes"
	"context"

	//nolint:gosec // BitTorrent uses SHA-1 for info hashes.
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/bencode"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/metainfo"
)

const (
	blockSize                     = 16 * 1024 // 16 kiB
	localMetadataExtensionID byte = 17
)

const (
	msgUnchoke    byte = 1
	msgInterested byte = 2
	msgBitfield   byte = 5
	msgRequest    byte = 6
	msgPiece      byte = 7
	msgExtension  byte = 20
)

// Dial opens a TCP connection to the given peer IP and performs the BitTorrent handshake.
func Dial(peerIP string, metaInfo *metainfo.MetaInfo) (net.Conn, string, error) {
	peerID := "PEERID12345678901234"

	content := make([]byte, 0, 68)
	content = append(content, byte(19))
	content = append(content, []byte("BitTorrent protocol")...)
	content = append(content, make([]byte, 8)...) // Reserved bytes
	content[len(content)-3] = 16                  // Set the extension protocol bit in the reserved bytes
	content = append(content, metaInfo.InfoHash...)
	content = append(content, []byte(peerID)...)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", peerIP)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()

	err = conn.SetDeadline(time.Now().Add(60 * time.Second))
	if err != nil {
		return nil, "", err
	}

	_, err = conn.Write(content)
	if err != nil {
		return nil, "", err
	}

	response := make([]byte, 68)
	_, err = io.ReadFull(conn, response)
	if err != nil {
		return nil, "", err
	}

	return conn, string(response[48:68]), nil
}

// ReadBitfield waits for a bitfield message from the peer indicating which pieces it has.
func ReadBitfield(conn net.Conn) error {
	// Wait for a bitfield message from the peer indicating which pieces it has
	messageID, _, err := readMessage(conn)
	if err != nil {
		return err
	}
	if messageID != msgBitfield {
		return fmt.Errorf("expected bitfield message, got message ID %d", messageID)
	}
	return nil
}

// Setup completes the post-handshake negotiation: sends Interested, and waits for Unchoke.
func Setup(conn net.Conn) error {
	// Send an interested message
	err := sendMessage(conn, msgInterested, nil)
	if err != nil {
		return err
	}

	// Wait until you receive an unchoke message back
	messageID, _, err := readMessage(conn)
	if err != nil {
		return err
	}

	if messageID != msgUnchoke {
		return fmt.Errorf("expected unchoke message, got message ID %d", messageID)
	}
	return nil
}

// DownloadPiece breaks the piece into blocks of 16 kiB and download.
func DownloadPiece(conn net.Conn, metaInfo *metainfo.MetaInfo, pieceIndex int) ([]byte, error) {
	if pieceIndex < 0 || pieceIndex >= len(metaInfo.InfoDict.PieceHashes) {
		return nil, fmt.Errorf("invalid piece index %d", pieceIndex)
	}

	pieceLength := min(metaInfo.InfoDict.PieceLength, metaInfo.InfoDict.Length-pieceIndex*metaInfo.InfoDict.PieceLength)
	piece := make([]byte, pieceLength)
	for blockIndex := 0; blockIndex*blockSize < pieceLength; blockIndex++ {
		// Break the piece into blocks of 16 kiB and send a request message for each block
		blockBegin := blockIndex * blockSize
		blockLength := min(blockSize, pieceLength-blockBegin)
		payload := make([]byte, 12)
		//nolint:gosec // BitTorrent protocol defines piece index, block begin, and block length as uint32.
		binary.BigEndian.PutUint32(payload[0:4], uint32(pieceIndex))
		binary.BigEndian.PutUint32(payload[4:8], uint32(blockBegin))
		//nolint:gosec // BitTorrent protocol defines block length as uint32.
		binary.BigEndian.PutUint32(payload[8:12], uint32(blockLength))

		err := sendMessage(conn, msgRequest, payload)
		if err != nil {
			return nil, err
		}

		// Wait for a piece message for each block you've requested
		messageID, piecePayload, err := readMessage(conn)
		if err != nil {
			return nil, err
		}
		if messageID != msgPiece {
			return nil, fmt.Errorf("expected piece message, got message ID %d", messageID)
		}
		if len(piecePayload) != 8+blockLength {
			return nil, fmt.Errorf("invalid piece payload length: expected %d, got %d", 8+blockLength, len(piecePayload))
		}

		messageIndex := int(binary.BigEndian.Uint32(piecePayload[0:4]))
		if messageIndex != pieceIndex {
			return nil, fmt.Errorf("invalid piece payload: expected piece index %d, got %d", pieceIndex, messageIndex)
		}
		messageBegin := int(binary.BigEndian.Uint32(piecePayload[4:8]))
		if messageBegin != blockBegin {
			return nil, fmt.Errorf("invalid piece payload: expected block begin %d, got %d", blockBegin, messageBegin)
		}
		if messageBegin+blockLength > pieceLength {
			return nil, fmt.Errorf("invalid piece payload: block exceeds piece length")
		}

		copy(piece[messageBegin:messageBegin+blockLength], piecePayload[8:8+blockLength])
	}
	// Verify that the SHA-1 hash of the piece matches the expected hash from the torrent file.
	//nolint:gosec // BitTorrent uses SHA-1 for info hashes.
	actual := sha1.Sum(piece)
	expect := []byte(metaInfo.InfoDict.PieceHashes[pieceIndex])
	if !bytes.Equal(actual[:], expect) {
		return nil, fmt.Errorf("piece %d failed hash verification", pieceIndex)
	}
	return piece, nil
}

// ExtensionHandshake performs the extension protocol handshake.
func ExtensionHandshake(conn net.Conn) (int, error) {
	// Receive the bitfield message
	_, _, err := readMessage(conn)
	if err != nil {
		return 0, err
	}

	payload := []byte{0} // extension message ID
	handshakeDict := map[string]interface{}{
		"m": map[string]interface{}{
			"ut_metadata": int(localMetadataExtensionID),
		},
	}
	encodedHandshake, err := bencode.EncodeBencode(handshakeDict)
	if err != nil {
		return 0, err
	}
	payload = append(payload, encodedHandshake...)
	err = sendMessage(conn, msgExtension, payload)
	if err != nil {
		return 0, err
	}

	receivedID, receivedPayload, err := readMessage(conn)
	if err != nil {
		return 0, err
	}
	if receivedID != msgExtension {
		return 0, fmt.Errorf("expected extension message, got message ID %d", receivedID)
	}
	if len(receivedPayload) < 1 || receivedPayload[0] != 0 {
		return 0, fmt.Errorf("invalid extension handshake response: expected extension message with ID 0")
	}

	responseDict, _, err := bencode.DecodeBencode(string(receivedPayload), 1)
	if err != nil {
		return 0, err
	}

	responseMap, ok := responseDict.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("invalid extension handshake response: expected a dictionary")
	}

	mValue, ok := responseMap["m"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("invalid extension handshake response: missing 'm' dictionary")
	}

	utMetadataID, ok := mValue["ut_metadata"].(int)
	if !ok {
		return 0, fmt.Errorf("invalid extension handshake response: missing 'ut_metadata' ID")
	}
	return utMetadataID, nil
}

// ExtensionMetadata requests the metadata using the extension protocol.
func ExtensionMetadata(conn net.Conn, extensionID int) (*metainfo.InfoDict, error) {
	//nolint:gosec // This is a command-line tool. We trust the user to provide a valid extension ID.
	payload := []byte{byte(extensionID)}
	metadataDict := map[string]interface{}{
		"msg_type": 0, // request
		"piece":    0,
	}
	encodedMetadata, err := bencode.EncodeBencode(metadataDict)
	if err != nil {
		return nil, err
	}
	payload = append(payload, encodedMetadata...)
	err = sendMessage(conn, msgExtension, payload)
	if err != nil {
		return nil, err
	}

	receivedID, receivedPayload, err := readMessage(conn)
	if err != nil {
		return nil, err
	}
	if receivedID != msgExtension {
		return nil, fmt.Errorf("expected extension message, got message ID %d", receivedID)
	}

	if len(receivedPayload) < 1 {
		return nil, fmt.Errorf("invalid metadata response: payload too short")
	}

	if receivedPayload[0] != localMetadataExtensionID {
		return nil, fmt.Errorf(
			"invalid metadata response: expected extension ID %d, got %d",
			localMetadataExtensionID,
			receivedPayload[0],
		)
	}

	receivedText := string(receivedPayload)

	_, start, err := bencode.DecodeBencode(receivedText, 1)
	if err != nil {
		return nil, err
	}

	infoDictRaw, _, err := bencode.DecodeBencode(receivedText, start)
	if err != nil {
		return nil, err
	}

	infoDictOk, ok := infoDictRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid metadata response: expected a dictionary")
	}

	infoDict, err := metainfo.ParseInfoDict(infoDictOk)
	if err != nil {
		return nil, err
	}
	return infoDict, nil
}

func readMessage(conn net.Conn) (byte, []byte, error) {
	for {
		lengthBuf := make([]byte, 4)
		_, err := io.ReadFull(conn, lengthBuf)
		if err != nil {
			return 0, nil, err
		}

		length := int(binary.BigEndian.Uint32(lengthBuf))
		if length == 0 {
			continue // Keep-alive message, read the next message
		}

		payload := make([]byte, length)
		_, err = io.ReadFull(conn, payload)
		if err != nil {
			return 0, nil, err
		}

		messageID := payload[0]
		return messageID, payload[1:], nil
	}
}

func sendMessage(conn net.Conn, messageID byte, payload []byte) error {
	length := 1 + len(payload)
	message := make([]byte, 4+length)
	//nolint:gosec // BitTorrent protocol defines message length as uint32.
	binary.BigEndian.PutUint32(message[0:4], uint32(length))
	message[4] = messageID
	copy(message[5:], payload)

	_, err := conn.Write(message)
	return err
}
