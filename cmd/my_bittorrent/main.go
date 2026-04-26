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

	var err error
	switch command {
	case "decode":
		err = cmdDecode()
	case "info":
		err = cmdInfo()
	case "peers":
		err = cmdPeers()
	case "handshake":
		err = cmdHandshake()
	case "download_piece":
		err = cmdDownloadPiece()
	case "download":
		err = cmdDownload()
	case "magnet_parse":
		err = cmdMagnetParse()
	case "magnet_handshake":
		err = cmdMagnetHandshake()
	case "magnet_info":
		err = cmdMagnetInfo()
	case "magnet_download_piece":
		err = cmdMagnetDownloadPiece()
	case "magnet_download":
		err = cmdMagnetDownload()
	default:
		err = fmt.Errorf("unknown command: %s", command)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
