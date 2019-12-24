package notify

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Xiol/tvhtc2/internal/pkg/media"
	"github.com/Xiol/tvhtc2/internal/pkg/notify/pushover"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Handler struct {
	pushoverToken string
}

func NewHandler(pushoverToken string) Handler {
	return Handler{
		pushoverToken: pushoverToken,
	}
}

func (h *Handler) createNotifications(entity *media.Entity) ([]Notifier, error) {
	if entity.Title == "" {
		return nil, nil
	}

	var notify []Notifier
	np, err := h.createPushoverNotifications(entity)
	if err != nil {
		return nil, err
	}

	notify = append(notify, np...)
	return notify, nil
}

func (h *Handler) createPushoverNotifications(entity *media.Entity) ([]Notifier, error) {
	var nconf []Config
	if err := viper.UnmarshalKey("notifications.pushover", &nconf); err != nil {
		return nil, fmt.Errorf("notify: error unmarshalling Pushover notification config: %s", err)
	}

	haveMatch := false
	defKey := ""
	defName := ""
	var notifiers []Notifier
	for _, conf := range nconf {
		if conf.Default {
			defName = conf.Name
			defKey = conf.Key
		}

		for _, rgx := range conf.Notify {
			testRegex, err := regexp.Compile(rgx)
			if err != nil {
				return nil, fmt.Errorf("notify: error compiling Pushover regex '%s': %s", rgx, err)
			}

			if testRegex.MatchString(strings.ToLower(entity.Title)) {
				notifiers = append(notifiers, pushover.NewMessage(h.pushoverToken, conf.Key, entity))
				haveMatch = true
				log.WithFields(log.Fields{
					"name":   conf.Name,
					"regexp": rgx,
					"title":  entity.Title,
				}).Debug("notify: adding pushover notification")
				break
			}
		}
	}

	if !haveMatch {
		notifiers = append(notifiers, pushover.NewMessage(h.pushoverToken, defKey, entity))
		log.WithFields(log.Fields{
			"name":  defName,
			"title": entity.Title,
		}).Debug("notify: no match for title, sending pushover to default user")
	}

	return notifiers, nil
}

func (h *Handler) DoNotifications(entity *media.Entity) error {
	var errs []error

	n, err := h.createNotifications(entity)
	if err != nil {
		return fmt.Errorf("notify: error creating notifications: %s", err)
	}

	for i := range n {
		err := n[i].Fire()
		if err != nil {
			log.WithField("error", err.Error()).Error("notify: error during notification: %s", err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notify: errors encountered: %s", strings.Join(h.strErrors(errs), ";"))
	}
	return nil
}

func (h *Handler) strErrors(errs []error) []string {
	s := make([]string, len(errs))
	for i := range errs {
		s[i] = errs[i].Error()
	}
	return s
}
