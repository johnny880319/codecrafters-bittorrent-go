// Package peer provides functions to extract information from torrent files and interact with peers.
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
	"net/http"
	"net/url"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/bencode"
)

// SendTrackerRequest sends a request to the tracker and prints the list of peers.
func SendTrackerRequest(torrentInfo *TorrentInfo) ([]string, error) {
	u, err := url.Parse(torrentInfo.TrackerURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("info_hash", string(torrentInfo.InfoHash))
	q.Set("peer_id", "PEERID12345678901234")
	q.Set("port", "6881")
	q.Set("uploaded", "0")
	q.Set("downloaded", "0")
	q.Set("left", fmt.Sprintf("%d", torrentInfo.Length))
	q.Set("compact", "1")
	u.RawQuery = q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	//nolint:gosec // BitTorrent clients intentionally connect to the tracker URL from the torrent file.
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parsePeers(body)
}

func parsePeers(body []byte) ([]string, error) {
	content, _, err := bencode.DecodeBencode(string(body), 0)
	if err != nil {
		return nil, err
	}

	peers, ok := content.(map[string]interface{})["peers"].(string)
	if !ok {
		return nil, fmt.Errorf("peers field is not a string")
	}

	var peerList []string
	for i := 0; i < len(peers); i += 6 {
		ip := peers[i : i+4]
		port := peers[i+4 : i+6]
		peerList = append(peerList, fmt.Sprintf("%d.%d.%d.%d:%d", ip[0], ip[1], ip[2], ip[3], int(port[0])*256+int(port[1])))
	}
	return peerList, nil
}

// OpenConnection opens a TCP connection to the given peer IP and performs the BitTorrent handshake.
func OpenConnection(peerIP string, torrentInfo *TorrentInfo) (net.Conn, string, error) {
	peerID := "PEERID12345678901234"

	content := make([]byte, 0, 68)
	content = append(content, byte(19))
	content = append(content, []byte("BitTorrent protocol")...)
	content = append(content, make([]byte, 8)...) // Reserved bytes
	content = append(content, torrentInfo.InfoHash...)
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

// ExchangePeerMessages performs the message exchange with the peer to download the pieces of the file.
// func ExchangePeerMessages(conn net.Conn, torrentInfo *TorrentInfo) ([]byte, error) {
// 	if err := SetupPeerConnection(conn); err != nil {
// 		return nil, err
// 	}

// 	pieces := make([]byte, 0, torrentInfo.Length)
// 	for pieceIndex := 0; pieceIndex*torrentInfo.PieceLength < torrentInfo.Length; pieceIndex++ {
// 		piece, err := DownloadPiece(conn, torrentInfo, pieceIndex)
// 		if err != nil {
// 			return nil, err
// 		}
// 		pieces = append(pieces, piece...)
// 	}
// 	return pieces, nil
// }

// SetupPeerConnection initiates the peer connection.
func SetupPeerConnection(conn net.Conn) error {
	// Wait for a bitfield message from the peer indicating which pieces it has
	messageID, _, err := readPeerMessage(conn)
	if err != nil {
		return err
	}
	if messageID != 5 { // Bitfield message ID is 5
		return fmt.Errorf("expected bitfield message, got message ID %d", messageID)
	}

	// Send an interested message
	err = sendPeerMessage(conn, 2, nil) // Interested message ID is 2
	if err != nil {
		return err
	}

	// Wait until you receive an unchoke message back
	messageID, _, err = readPeerMessage(conn)
	if err != nil {
		return err
	}
	if messageID != 1 { // Unchoke message ID is 1
		return fmt.Errorf("expected unchoke message, got message ID %d", messageID)
	}
	return nil
}

// DownloadPiece breaks the piece into blocks of 16 kiB and download.
func DownloadPiece(conn net.Conn, torrentInfo *TorrentInfo, pieceIndex int) ([]byte, error) {
	pieceLength := min(torrentInfo.PieceLength, torrentInfo.Length-pieceIndex*torrentInfo.PieceLength)
	piece := make([]byte, pieceLength)
	for blockIndex := 0; blockIndex*16*1024 < pieceLength; blockIndex++ {
		// Break the piece into blocks of 16 kiB (16 * 1024 bytes) and send a request message for each block
		blockBegin := blockIndex * 16 * 1024
		blockLength := min(16*1024, pieceLength-blockBegin)
		payload := make([]byte, 12)
		//nolint:gosec // BitTorrent protocol defines piece index, block begin, and block length as uint32.
		binary.BigEndian.PutUint32(payload[0:4], uint32(pieceIndex))
		binary.BigEndian.PutUint32(payload[4:8], uint32(blockBegin))
		//nolint:gosec // BitTorrent protocol defines block length as uint32.
		binary.BigEndian.PutUint32(payload[8:12], uint32(blockLength))

		err := sendPeerMessage(conn, 6, payload) // Request message ID is 6
		if err != nil {
			return nil, err
		}

		// Wait for a piece message for each block you've requested
		messageID, piecePayload, err := readPeerMessage(conn)
		if err != nil {
			return nil, err
		}
		if messageID != 7 { // Piece message ID is 7
			return nil, fmt.Errorf("expected piece message, got message ID %d", messageID)
		}

		messageBegin := int(binary.BigEndian.Uint32(piecePayload[4:8]))
		copy(piece[messageBegin:messageBegin+blockLength], piecePayload[8:8+blockLength])
	}
	// Verify that the SHA-1 hash of the piece matches the expected hash from the torrent file.
	//nolint:gosec // BitTorrent uses SHA-1 for info hashes.
	actual := sha1.Sum(piece)
	expect := []byte(torrentInfo.PieceHashes[pieceIndex])
	if !bytes.Equal(actual[:], expect) {
		return nil, fmt.Errorf("piece %d failed hash verification", pieceIndex)
	}
	return piece, nil
}

func readPeerMessage(conn net.Conn) (byte, []byte, error) {
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

func sendPeerMessage(conn net.Conn, messageID byte, payload []byte) error {
	length := 1 + len(payload)
	message := make([]byte, 4+length)
	//nolint:gosec // BitTorrent protocol defines message length as uint32.
	binary.BigEndian.PutUint32(message[0:4], uint32(length))
	message[4] = messageID
	copy(message[5:], payload)

	_, err := conn.Write(message)
	return err
}
