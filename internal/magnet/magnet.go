// Package magnet provides functionality for parsing and handling magnet links.
package magnet

import (
	"fmt"
	"strings"
)

// Parse extracts the tracker URL and info hash from a magnet URI.
func Parse(magnetURI string) (trackerURL string, infoHash string, err error) {
	trackerURLStart := strings.Index(magnetURI, "tr=") + len("tr=")
	if trackerURLStart < len("tr=") {
		return "", "", fmt.Errorf("invalid magnet URI: missing 'tr=' parameter")
	}
	trackerURLEnd := strings.Index(magnetURI[trackerURLStart:], "&")
	if trackerURLEnd == -1 {
		trackerURLEnd = len(magnetURI)
	} else {
		trackerURLEnd += trackerURLStart
	}
	trackerURL = magnetURI[trackerURLStart:trackerURLEnd]

	infoHashStart := strings.Index(magnetURI, "xt=urn:btih:") + len("xt=urn:btih:")
	if infoHashStart < len("xt=urn:btih:") {
		return "", "", fmt.Errorf("invalid magnet URI: missing 'xt=urn:btih:' prefix")
	}
	infoHash = magnetURI[infoHashStart : infoHashStart+40]

	return trackerURL, infoHash, nil
}
