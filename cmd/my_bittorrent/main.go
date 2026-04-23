// Package main is the entry point for the BitTorrent client.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: my_bittorrent <command> [args...]")
		os.Exit(1)
	}
	command := os.Args[1]

	switch command {
	case "decode":
		cmdDecode()
	case "info":
		cmdInfo()
	case "peers":
		cmdPeers()
	case "handshake":
		cmdHandshake()
	case "download_piece":
		cmdDownloadPiece()
	case "download":
		cmdDownload()
	default:
		//nolint:gosec // CLI tool, stderr output is not an XSS vector
		fmt.Fprintln(os.Stderr, "Unknown command: "+command)
		os.Exit(1)
	}
}
