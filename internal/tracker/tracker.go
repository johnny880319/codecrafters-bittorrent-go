// Package tracker implements the logic to send a request to the tracker and parse the response.
package tracker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/bencode"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/metainfo"
)

// GetPeers sends a request to the tracker and returns the list of peers.
func GetPeers(metaInfo *metainfo.MetaInfo) ([]string, error) {
	u, err := url.Parse(metaInfo.TrackerURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("info_hash", string(metaInfo.InfoHash))
	q.Set("peer_id", "PEERID12345678901234")
	q.Set("port", "6881")
	q.Set("uploaded", "0")
	q.Set("downloaded", "0")
	q.Set("left", fmt.Sprintf("%d", metaInfo.InfoDict.Length))
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

	contentDict, ok := content.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("tracker response is not a dictionary")
	}

	peers, ok := contentDict["peers"].(string)
	if !ok {
		return nil, fmt.Errorf("peers field is not a string")
	}
	if len(peers)%6 != 0 {
		return nil, fmt.Errorf("invalid peers field length")
	}

	var peerList []string
	for i := 0; i < len(peers); i += 6 {
		ip := peers[i : i+4]
		port := peers[i+4 : i+6]
		peerList = append(peerList, fmt.Sprintf("%d.%d.%d.%d:%d", ip[0], ip[1], ip[2], ip[3], int(port[0])*256+int(port[1])))
	}
	return peerList, nil
}
