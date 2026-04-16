package peer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/parse"
)

// SendTrackerRequest sends a request to the tracker and prints the list of peers.
func SendTrackerRequest(trackerURL string, infoHash []byte) error {
	u, err := url.Parse(trackerURL)
	if err != nil {
		return err
	}
	fmt.Println(string(infoHash))
	q := u.Query()
	q.Set("info_hash", string(infoHash))
	q.Set("peer_id", "PEERID12345678901234")
	q.Set("port", "6881")
	q.Set("uploaded", "0")
	q.Set("downloaded", "0")
	q.Set("left", "0")
	q.Set("compact", "1")
	u.RawQuery = q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	client := &http.Client{}

	//nolint:gosec // BitTorrent clients intentionally connect to the tracker URL from the torrent file.
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	content, _, err := parse.DecodeBencode(string(body), 0)
	if err != nil {
		return err
	}

	peers, ok := content.(map[string]interface{})["peers"].(string)
	if !ok {
		return fmt.Errorf("peers field is not a string")
	}

	for i := 0; i < len(peers); i += 6 {
		ip := peers[i : i+4]
		port := peers[i+4 : i+6]
		fmt.Printf("%d.%d.%d.%d:%d\n", ip[0], ip[1], ip[2], ip[3], int(port[0])*256+int(port[1]))
	}
	return nil
}
