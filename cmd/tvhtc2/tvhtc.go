package main

import (
	"strings"

	"github.com/Xiol/tvhtc2/internal/pkg/config"
	"github.com/Xiol/tvhtc2/internal/pkg/notify"
	"github.com/Xiol/tvhtc2/internal/pkg/transcoder"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	log.SetLevel(log.DebugLevel)
	fmt := log.TextFormatter{
		DisableTimestamp: true,
	}
	log.SetFormatter(&fmt)
	log.Warning("TVHTC2 starting up...")

	if err := config.InitConfig(); err != nil {
		log.Fatal(err.Error())
	}

	notificationHandler := notify.NewHandler(viper.GetString("pushover.app_token"))
	t, err := transcoder.New(transcoder.Config{
		AudioArgs:  strings.Split(viper.GetString("transcoding.audio_config"), " "),
		VideoArgs:  strings.Split(viper.GetString("transcoding.video_config"), " "),
		OnlySD:     viper.GetBool("transcoding.only_sd"),
		Keep:       viper.GetBool("transcoding.keep"),
		TrimPath:   viper.GetString("transcoding.trim_path"),
		StatePath:  viper.GetString("state_path"),
		SocketPath: viper.GetString("socket_path"),
	}, notificationHandler)
	if err != nil {
		log.Fatalf("error initialising transcoder: %s", err)
	}

	defer t.Close()
	if err := t.Do(); err != nil {
		log.WithError(err).Error("error during transcoder startup")
	}
}
