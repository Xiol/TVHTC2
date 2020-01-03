package main

import (
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
	t, err := transcoder.New(notificationHandler)
	if err != nil {
		log.Fatalf("error initialising transcoder: %s", err)
	}

	defer t.Close()
	if err := t.Do(); err != nil {
		log.WithError(err).Error("error during transcoder startup")
	}
}
