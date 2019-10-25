package state

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/Xiol/tvhtc2/internal/pkg/media"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

const defaultJobChannelSize int = 64

type Job struct {
	ID      string
	Details *media.Details
}

type State struct {
	sync.Mutex
	Pending map[string]media.Details `json:"pending"`
	JobCh   chan *Job                `json:"-"`

	path string
}

func NewState(path string) (*State, error) {
	s := State{
		Pending: make(map[string]media.Details),
		JobCh:   make(chan *Job, defaultJobChannelSize),
		path:    path,
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	log.WithField("path", path).Debug("state: initialised state using file")

	return &s, nil
}

func (s *State) load() error {
	s.Lock()
	defer s.Unlock()

	rb, err := ioutil.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s.save()
		}
		return fmt.Errorf("state: %s", err)
	}

	if err := json.Unmarshal(rb, s); err != nil {
		return fmt.Errorf("state: error unmarshalling: %s", err)
	}

	l := len(s.Pending)
	if l > defaultJobChannelSize {
		if l > 4096 {
			log.Fatalf("state: pending jobs >4096 (%s)?", l)
		}
		log.Debug("state: resizing job channel")
		s.JobCh = make(chan *Job, l*2)
	}

	for id, details := range s.Pending {
		log.WithFields(log.Fields{
			"id":    id,
			"title": details.Title,
		}).Info("state: adding pending job")
		s.JobCh <- &Job{id, &details}
	}

	log.WithField("pending_count", len(s.Pending)).Info("state: loaded state from disk")

	return nil
}

func (s *State) save() error {
	log.Debug("state: saving state to disk")

	jout, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("state: error marshalling: %s", err)
	}

	err = ioutil.WriteFile(s.path, jout, 0640)
	if err != nil {
		return fmt.Errorf("state: error writing state: %s", err)
	}

	return nil
}

func (s *State) Add(d media.Details) error {
	s.Lock()
	defer s.Unlock()
	id := uuid.Must(uuid.NewUUID()).String()
	s.Pending[id] = d
	s.JobCh <- &Job{id, &d}

	log.WithFields(log.Fields{
		"title": d.Title,
		"id":    id,
	}).Debug("state: appending new entity to state")

	return s.save()
}

func (s *State) Done(id string) error {
	s.Lock()
	defer s.Unlock()

	log.WithFields(log.Fields{
		"title": s.Pending[id].Title,
		"id":    id,
	}).Debug("state: removing entity from state")

	delete(s.Pending, id)
	return s.save()
}
