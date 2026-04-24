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

const blockSize = 16 * 1024 // 16 kiB
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

// Setup completes the post-handshake negotiation: reads the peer's bitfield,
// sends Interested, and waits for Unchoke.
func Setup(conn net.Conn) error {
	// Wait for a bitfield message from the peer indicating which pieces it has
	messageID, _, err := readMessage(conn)
	if err != nil {
		return err
	}
	if messageID != msgBitfield {
		return fmt.Errorf("expected bitfield message, got message ID %d", messageID)
	}

	// Send an interested message
	err = sendMessage(conn, msgInterested, nil)
	if err != nil {
		return err
	}

	// Wait until you receive an unchoke message back
	messageID, _, err = readMessage(conn)
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
	pieceLength := min(metaInfo.PieceLength, metaInfo.Length-pieceIndex*metaInfo.PieceLength)
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

		messageBegin := int(binary.BigEndian.Uint32(piecePayload[4:8]))
		copy(piece[messageBegin:messageBegin+blockLength], piecePayload[8:8+blockLength])
	}
	// Verify that the SHA-1 hash of the piece matches the expected hash from the torrent file.
	//nolint:gosec // BitTorrent uses SHA-1 for info hashes.
	actual := sha1.Sum(piece)
	expect := []byte(metaInfo.PieceHashes[pieceIndex])
	if !bytes.Equal(actual[:], expect) {
		return nil, fmt.Errorf("piece %d failed hash verification", pieceIndex)
	}
	return piece, nil
}

// ExtensionHandshake performs the extension protocol handshake.
func ExtensionHandshake(conn net.Conn) error {
	payload := []byte{0} // extension message ID
	handshakeDict := map[string]interface{}{
		"m": map[string]interface{}{
			"ut_metadata": 16,
		},
	}
	encodedHandshake, err := bencode.EncodeBencode(handshakeDict)
	if err != nil {
		return err
	}
	payload = append(payload, encodedHandshake...)
	err = sendMessage(conn, msgExtension, payload)
	if err != nil {
		return err
	}
	return nil
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
