package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"

	"github.com/honeynet/ochi/backend"
)

//go:embed public/*
var public embed.FS

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config file")
	flag.Parse()

	content, err := fs.Sub(public, "public")
	if err != nil {
		log.Fatal(err)
	}

	srv, err := backend.NewServer(content, *configPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
