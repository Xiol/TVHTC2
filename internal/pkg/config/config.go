package config

import (
	"fmt"
	"regexp"

	"github.com/Xiol/tvhtc2/internal/pkg/notify"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func InitConfig() error {
	viper.SetConfigName("tvhtc2")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/tvhtc2/")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("config: error reading config: %s", err)
	}

	if err := validateRegexps(); err != nil {
		log.Fatalf(err.Error())
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Infof("config: file at '%s' changed", e.Name)
		if err := validateRegexps(); err != nil {
			log.Fatalf(err.Error())
		}
		log.Info("config: reloaded configuration")
	})

	return nil
}

func validateRegexps() error {
	log.Debug("config: validating regexps...")

	var nconf []notify.Config
	if err := viper.UnmarshalKey("notifications.pushover", &nconf); err != nil {
		return fmt.Errorf("config: error unmarshalling Pushover notification config: %s", err)
	}

	count := 0
	for _, conf := range nconf {
		for _, rgx := range conf.Notify {
			_, err := regexp.Compile(rgx)
			if err != nil {
				return fmt.Errorf("config: regex '%s' for user '%s' did not compile, please check config: %s", rgx, conf.Name, err)
			}
			count++
		}
	}

	log.Debugf("config: regex validation ok, count %d", count)
	return nil
}
