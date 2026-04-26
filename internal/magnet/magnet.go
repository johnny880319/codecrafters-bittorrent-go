// Package magnet provides functionality for parsing and handling magnet links.
package magnet

import (
	"fmt"
	"net/url"
	"strings"
)

// Parse extracts the tracker URL and info hash from a magnet URI.
func Parse(magnetURI string) (trackerURL string, infoHash string, err error) {
	u, err := url.Parse(magnetURI)
	if err != nil {
		return "", "", fmt.Errorf("invalid magnet URI: %w", err)
	}

	values := u.Query()

	trackerURL = values.Get("tr")
	if trackerURL == "" {
		return "", "", fmt.Errorf("invalid magnet URI: missing 'tr' parameter")
	}

	infoHash = strings.TrimPrefix(values.Get("xt"), "urn:btih:")
	if infoHash == "" {
		return "", "", fmt.Errorf("invalid magnet URI: missing 'xt' parameter")
	}

	return trackerURL, infoHash, nil
}
