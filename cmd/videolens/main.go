package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/alexandrmotologa/videolens/internal/cli"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "serve":
		serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
		port := serveCmd.Int("port", 8850, "HTTP port to listen on")
		_ = serveCmd.Parse(os.Args[2:])

		if err := cli.RunServe(*port); err != nil {
			fmt.Fprintf(os.Stderr, "Error running server: %v\n", err)
			os.Exit(1)
		}

	case "check":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: videolens check <url-or-file>")
			os.Exit(1)
		}
		target := os.Args[2]
		if err := cli.RunCheck(target); err != nil {
			fmt.Fprintf(os.Stderr, "Check failed: %v\n", err)
			os.Exit(1)
		}

	case "faststart":
		fastCmd := flag.NewFlagSet("faststart", flag.ExitOnError)
		outFlag := fastCmd.String("o", "", "Output optimized file path")
		_ = fastCmd.Parse(os.Args[2:])

		args := fastCmd.Args()
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: videolens faststart <input.mp4> [-o <output.mp4>]")
			os.Exit(1)
		}
		inputFile := args[0]
		if err := cli.RunFastStart(inputFile, *outFlag); err != nil {
			fmt.Fprintf(os.Stderr, "FastStart remediation failed: %v\n", err)
			os.Exit(1)
		}

	case "abr":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: videolens abr <master-playlist-url-or-file>")
			os.Exit(1)
		}
		target := os.Args[2]
		if err := cli.RunABR(target); err != nil {
			fmt.Fprintf(os.Stderr, "ABR ladder validation failed: %v\n", err)
			os.Exit(1)
		}

	case "version", "--version", "-v":
		fmt.Printf("VideoLens v%s\n", cli.Version)

	case "help", "--help", "-h":
		printUsage()

	default:
		// If argument looks like a URL or file, run check by default
		if err := cli.RunCheck(command); err != nil {
			fmt.Fprintf(os.Stderr, "Unknown command or check failed: %v\n", err)
			printUsage()
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Println(`VideoLens: Streaming Media Diagnostics Workbench & MP4 Box Profiler

Usage:
  videolens serve [--port 8850]                    Start embedded web studio server
  videolens check <url-or-file>                    Inspect HLS, DASH, or MP4 container
  videolens faststart <input.mp4> [-o <out.mp4>]   Relocate moov atom to head of MP4
  videolens abr <master-playlist>                  Audit ABR ladder and Bits Per Pixel
  videolens version                                Show version

Examples:
  videolens serve --port 8850
  videolens check https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8
  videolens check video.mp4
  videolens faststart raw_capture.mp4 -o web_ready.mp4
  videolens abr master.m3u8`)
}
