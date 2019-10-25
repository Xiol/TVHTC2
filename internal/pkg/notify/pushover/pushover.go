package pushover

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Xiol/tvhtc2/internal/pkg/media"
	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
)

const endpoint string = "https://api.pushover.net/1/messages.json"

type Priority int

const (
	PriorityNoNotification      Priority = -2
	PrioritySilent              Priority = -1
	PriorityNormal              Priority = 0
	PriorityHighPriority        Priority = 1
	PriorityRequireConfirmation Priority = 2
)

type Response struct {
	Status  int    `json:"status"`
	Request string `json:"request"`
}

type Message struct {
	User     string
	Subject  string
	Body     string
	Priority Priority
	ApiToken string
}

func NewMessage(apiToken, user string, entity *media.Entity) Message {
	var subj string
	if entity.Ok() {
		subj = fmt.Sprintf("New Recording: %s (%s)", entity.Title, entity.Channel)
	} else {
		subj = fmt.Sprintf("Failed Recording: %s (%s)", entity.Title, entity.Channel)
	}

	sb := strings.Builder{}
	sb.WriteString(strings.TrimSpace(entity.Description))
	sb.WriteString("\n\n")
	if entity.IsTranscodable() {
		sb.WriteString(fmt.Sprintf("Transcode completed in %d minutes, size change %s->%s. Path: %s",
			entity.Stats.Duration.Round(time.Minute), humanize.IBytes(entity.Stats.InitialSizeBytes),
			humanize.IBytes(entity.Stats.EndSizeBytes), entity.Path))
	} else {
		sb.WriteString(fmt.Sprintf("Skipped transcoding. Size %s. Path: %s",
			humanize.IBytes(entity.Stats.InitialSizeBytes), entity.Path))
	}

	return Message{
		User:     user,
		Subject:  subj,
		Body:     sb.String(),
		Priority: PriorityNormal,
		ApiToken: apiToken,
	}
}

func (m Message) values() url.Values {
	v := url.Values{}
	v.Add("token", m.ApiToken)
	v.Add("user", m.User)
	v.Add("priority", strconv.Itoa(int(m.Priority)))
	v.Add("timestamp", strconv.Itoa(int(time.Now().Unix())))
	v.Add("message", m.Body)
	v.Add("title", m.Subject)
	return v
}

func (m Message) Fire() error {
	payload := m.values()

	resp, err := http.PostForm(endpoint, payload)
	if err != nil {
		return fmt.Errorf("pushover: error sending notification: %s", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("pushover: unable to read response body: %s", err)
	}

	if resp.StatusCode > 200 && resp.StatusCode < 500 {
		return fmt.Errorf("pushover: bad status code %d from Pushover API, response body: %s",
			resp.StatusCode, body)
	}

	if resp.StatusCode >= 500 {
		return fmt.Errorf("pushover: status code %d from Pushover API indicates temporary failure, but not retrying")
	}

	pr := Response{}
	err = json.Unmarshal(body, &pr)
	if err != nil {
		log.WithField("body", body).Error("pushover: failed to unmarshal response from API")
		return fmt.Errorf("pushover: could not unmarshal response from Pushover API: %s", err)
	}

	if pr.Status != 1 {
		return fmt.Errorf("pushover: API status was %d, expected 1, notification may not have been sent", pr.Status)
	}

	log.WithField("user", m.User).Info("pushover: notification sent")
	log.WithField("message", m.Body).Debug("pushover: message body")
	return nil
}
