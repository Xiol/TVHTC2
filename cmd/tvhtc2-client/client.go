package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/Xiol/tvhtc2/internal/pkg/config"
	"github.com/Xiol/tvhtc2/internal/pkg/media"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	log.SetLevel(log.FatalLevel)
	log.Warning("TVHTC2 client initialising...")

	if err := config.InitConfig(); err != nil {
		log.Fatal(err.Error())
	}

	var path = flag.String("path", "", "path to file")
	var channel = flag.String("channel", "", "channel")
	var title = flag.String("title", "", "programme title")
	var status = flag.String("status", "", "status of recording")
	var description = flag.String("description", "", "description of programme")
	flag.Parse()

	if *path == "" {
		log.Fatal("missing path")
	}

	if *channel == "" {
		log.Fatal("missing channel")
	}

	if *title == "" {
		log.Fatal("missing title")
	}

	if *status == "" {
		log.Fatal("missing status")
	}

	if *description == "" {
		log.Fatal("missing description")
	}

	details := media.Details{
		Path:        *path,
		Channel:     *channel,
		Title:       *title,
		Status:      *status,
		Description: *description,
	}

	payload, err := json.Marshal(details)
	if err != nil {
		log.Fatalf("failed to marshal programme details: %s", err)
	}

	c, err := net.Dial("unix", viper.GetString("socket_path"))
	if err != nil {
		log.Fatalf("failed to dial TVHTC2 socket: %s", err)
	}

	i, err := c.Write(payload)
	if err != nil {
		log.Fatalf("failed to write payload to socket after %d bytes: %s", i, err)
	}

	fmt.Printf("ok\n")
	os.Exit(0)
}
