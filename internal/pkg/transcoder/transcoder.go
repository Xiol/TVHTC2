package transcoder

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/Xiol/tvhtc2/internal/pkg/media"
	"github.com/Xiol/tvhtc2/internal/pkg/notify"
	"github.com/Xiol/tvhtc2/internal/pkg/state"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	AudioArgs  []string
	VideoArgs  []string
	OnlySD     bool
	Keep       bool
	TrimPath   string
	StatePath  string
	SocketPath string
}

type Transcoder struct {
	binaryPath          string
	notificationHandler notify.Handler
	config              Config
	state               *state.State
	incCloseCh          chan struct{}
	trnCloseCh          chan struct{}
}

func New(conf Config, notificationHandler notify.Handler, options ...func(*Transcoder)) (Transcoder, error) {
	t := Transcoder{
		config:              conf,
		notificationHandler: notificationHandler,
		incCloseCh:          make(chan struct{}),
		trnCloseCh:          make(chan struct{}),
	}

	for _, opt := range options {
		opt(&t)
	}

	var err error
	if t.state, err = state.NewState(conf.StatePath); err != nil {
		return t, err
	}

	return t, nil
}

func BinaryPath(path string) func(*Transcoder) {
	return func(t *Transcoder) {
		t.binaryPath = path
	}
}

func (t *Transcoder) Close() {
	t.incCloseCh <- struct{}{}
	t.trnCloseCh <- struct{}{}
}

// Do will start handling jobs. This function blocks.
func (t *Transcoder) Do() error {
	if err := t.listen(); err != nil {
		return err
	}
	log.Info("transcoder: ready for jobs")
	t.transcodeHandler()
	return nil
}

func (t *Transcoder) listen() error {
	if err := os.RemoveAll(t.config.SocketPath); err != nil {
		return fmt.Errorf("transcoder: error removing old socket: %s", err)
	}

	listener, err := net.Listen("unix", t.config.SocketPath)
	if err != nil {
		return fmt.Errorf("transcoder: error listening at unix:%s: %s", t.config.SocketPath, err)
	}

	go func(l net.Listener, closeCh chan struct{}) {
		defer l.Close()
		for {
			select {
			case <-closeCh:
				return
			default:
				conn, err := l.Accept()
				if err != nil {
					log.WithError(err).Error("transcoder: accept error")
				}
				go t.incomingHandler(conn)
			}
		}
	}(listener, t.incCloseCh)
	return nil
}

func (t *Transcoder) incomingHandler(conn net.Conn) {
	data, err := ioutil.ReadAll(conn)
	if err != nil {
		log.WithError(err).Error("transcoder: socket read error")
		return
	}

	var details media.Details
	err = json.Unmarshal(data, &details)
	if err != nil {
		log.WithError(err).Error("transcoder: failed to unmarshal media details: %s", err)
		return
	}

	if err := t.state.Add(details); err != nil {
		log.WithError(err).Error("transcoder: failed to add media entity to state")
		return
	}
	return
}

func (t *Transcoder) transcodeHandler() {
	for {
		select {
		case <-t.trnCloseCh:
			return
		case job := <-t.state.JobCh:
			e, err := media.NewEntity(*job.Details, media.Config{
				OnlySD:    t.config.OnlySD,
				AudioArgs: t.config.AudioArgs,
				VideoArgs: t.config.VideoArgs,
			})
			if err != nil {
				log.WithError(err).Error("transcoder: error creating entity")
			}

			if err := e.Transcode(); err != nil {
				log.WithError(err).Error("transcoder: error during transcode")
				t.notify(e)
				return
			}

			if err := t.state.Done(job.ID); err != nil {
				log.WithError(err).Error("transcoder: failed to mark job as done: %s", err)
				t.notify(e)
				return
			}

			t.notify(e)
		}
	}
}

func (t *Transcoder) notify(e *media.Entity) {
	if err := t.notificationHandler.DoNotifications(e); err != nil {
		log.WithError(err).Error("transcoder: error when doing notifications")
	}
}
