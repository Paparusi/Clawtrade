package main

import (
	"fmt"
	"os"

	"github.com/clawtrade/clawtrade/internal"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: clawtrade <command>")
		fmt.Println("Commands: version, serve")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("Clawtrade %s\n", internal.Version)
	case "serve":
		fmt.Println("Starting Clawtrade server...")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
