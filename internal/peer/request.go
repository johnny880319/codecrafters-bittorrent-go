package peer

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/parse"
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
	content, _, err := parse.DecodeBencode(string(body), 0)
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

// PerformHandshake performs the BitTorrent handshake with the given peer and returns the peer ID from the response.
func PerformHandshake(peerIP string, torrentInfo *TorrentInfo) (string, error) {
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
		return "", err
	}
	defer func() {
		err = conn.Close()
	}()
	if err != nil {
		return "", err
	}

	err = conn.SetDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return "", err
	}

	_, err = conn.Write(content)
	if err != nil {
		return "", err
	}

	response := make([]byte, 68)
	_, err = io.ReadFull(conn, response)
	if err != nil {
		return "", err
	}

	return string(response[48:68]), nil
}
