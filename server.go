package main

import (
	"embed"
	"io/fs"
	"log"

	"github.com/honeynet/ochi/backend"
)

//go:embed public
var public embed.FS

func main() {
	content, err := fs.Sub(public, "public")
	if err != nil {
		log.Fatal(err)
	}
	srv := backend.NewServer(content)
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
