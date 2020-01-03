package media

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Xiol/tvhtc2/internal/pkg/renamer"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/vansante/go-ffprobe"
)

type Stats struct {
	Duration         time.Duration `json:"duration"`
	InitialSizeBytes uint64        `json:"initial_size_bytes"`
	EndSizeBytes     uint64        `json:"end_size_bytes"`
	CommandStdout    []byte        `json:"command_stdout"`
}

type Details struct {
	Path        string `json:"path"`
	Channel     string `json:"channel"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type Type int

const (
	MEDIA_VIDEO Type = 1 + iota
	MEDIA_H264_VIDEO
	MEDIA_AUDIO
	MEDIA_UNKNOWN
)

type Entity struct {
	Details
	DestPath         string `json:"dest_path"`
	Media            Type   `json:"type"`
	Stats            Stats  `json:"stats"`
	TranscodeSuccess bool   `json:"transcode_success"`

	renamer       renamer.Renamer
	skipTranscode bool
	basename      string
	tmpfile       string
}

func NewEntity(details Details) (*Entity, error) {
	if details.Path == "" {
		return nil, fmt.Errorf("media: path must not be empty")
	}

	e := &Entity{
		Details:  details,
		Stats:    Stats{},
		DestPath: details.Path,
		renamer:  renamer.NewRenamer(),
	}
	e.tempFilename()
	return e, nil
}

func (e *Entity) detectMediaType() error {
	data, err := ffprobe.GetProbeData(e.Path, 3*time.Second)
	if err != nil {
		return fmt.Errorf("media: error getting probe data: %s", err)
	}

	vidStream := data.GetFirstVideoStream()
	if vidStream == nil {
		audioStream := data.GetFirstAudioStream()
		if audioStream == nil {
			e.Media = MEDIA_UNKNOWN
			return fmt.Errorf("media: found no audio or video streams in file")
		}
		return e.detectAudioOnly(audioStream)
	}
	return e.detectVideo(vidStream)
}

func (e *Entity) detectAudioOnly(stream *ffprobe.Stream) error {
	log.WithFields(log.Fields{
		"codec":    stream.CodecName,
		"type":     "audio",
		"filename": e.basename,
	}).Info("media: detected audio file codec")

	if stream.CodecName == "mp3" {
		e.skipTranscode = true
	}

	e.DestPath = strings.Replace(e.DestPath, filepath.Ext(e.Path), ".mp3", -1)
	e.Media = MEDIA_AUDIO
	return nil
}

func (e *Entity) detectVideo(stream *ffprobe.Stream) error {
	log.WithFields(log.Fields{
		"filename": e.basename,
		"codec":    stream.CodecName,
		"type":     "video",
	}).Info("media: detected video codec")

	switch stream.CodecName {
	case "h264":
		e.Media = MEDIA_H264_VIDEO
		if viper.GetBool("transcoding.only_sd") {
			log.Info("media: skipping transcode, only_sd is set")
			e.skipTranscode = true
		}
	default:
		e.Media = MEDIA_VIDEO
	}
	return nil
}

func (e *Entity) tempFilename() {
	dir := filepath.Dir(e.Path)
	var ext string
	if e.Media == MEDIA_VIDEO || e.Media == MEDIA_H264_VIDEO {
		ext = filepath.Ext(e.Path)
		if ext == "" {
			ext = ".mkv"
		}
	}
	if e.Media == MEDIA_AUDIO {
		ext = ".mp3"
	}

	e.tmpfile = filepath.Join(dir, uuid.New().String()+ext)
	log.WithField("path", e.tmpfile).Debug("media: temporary path for encoding media")
}

func (e *Entity) ffmpegArgs() []string {
	var args string
	if e.Media == MEDIA_VIDEO || e.Media == MEDIA_H264_VIDEO {
		args = viper.GetString("transcoding.video_args")
	} else {
		args = viper.GetString("transcoding.audio_args")
	}
	return strings.Split(args, " ")
}

func (e *Entity) rename() error {
	newPath := e.renamer.Rename(e.DestPath)
	log.WithFields(log.Fields{
		"new_path": newPath,
		"old_path": e.Path,
	}).Debug("rename results")

	if viper.GetBool("transcoding.skip_rename") {
		log.Debug("skip_rename is set, not perfoming full renaming")
		os.Rename(e.tmpfile, e.DestPath)
		return nil
	}
	e.DestPath = newPath

	// Create the renamed directory, if needed
	dir, _ := filepath.Split(e.DestPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.Mkdir(dir, 0755); err != nil {
			return fmt.Errorf("media: unable to create directory %s: %s", dir, err)
		}
	}

	log.WithFields(log.Fields{
		"src":  e.Path,
		"dest": e.DestPath,
	}).Info("media: renaming transcoded file")
	os.Rename(e.tmpfile, e.DestPath)

	return nil
}

func (e *Entity) cleanup() error {
	// If we're skipping renames, and the media is a video, then we'll have nothing
	// to clean up, return early.
	if viper.GetBool("transcoding.skip_rename") && e.Media != MEDIA_AUDIO {
		return nil
	}

	log.WithField("path", e.Path).Info("media: removing original file")
	if err := os.Remove(e.Path); err != nil {
		return fmt.Errorf("media: error removing original file: %s", err)
	}

	dir, _ := filepath.Split(e.Path)
	if e.dirEmpty(dir) {
		log.WithField("directory", dir).Info("media: directory empty, cleaning up")
		if err := os.Remove(dir); err != nil {
			return fmt.Errorf("media: error removing empty directory: %s", err)
		}
	}

	return nil
}

func (e *Entity) dirEmpty(dir string) bool {
	f, err := os.Open(dir)
	if err != nil {
		log.WithError(err).Error("error opening directory for read: %s", err)
		return false
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true
	}
	return false
}

func (e *Entity) abort() error {
	return os.Remove(e.tmpfile)
}

func (e *Entity) getSizeBytes(path string) uint64 {
	stat, err := os.Stat(path)
	if err != nil {
		return uint64(0)
	}

	return uint64(stat.Size())
}

func (e *Entity) SourcePath() string {
	return e.Path
}

func (e *Entity) DestinationPath() string {
	return e.DestPath
}

func (e *Entity) IsTranscodable() bool {
	return !e.skipTranscode
}

func (e *Entity) Ok() bool {
	return e.Status == "OK" && (e.skipTranscode || e.TranscodeSuccess)
}

func (e *Entity) Transcode() error {
	e.basename = filepath.Base(e.Details.Path)

	log.WithFields(log.Fields{
		"path":    e.Details.Path,
		"channel": e.Details.Channel,
		"title":   e.Details.Title,
		"status":  e.Details.Status,
	}).Info("media: new entity")

	e.Stats.InitialSizeBytes = e.getSizeBytes(e.Details.Path)

	err := e.detectMediaType()
	if err != nil {
		return err
	}

	if e.skipTranscode {
		log.WithFields(log.Fields{
			"filename": e.basename,
		}).Info("media: skipping transcode for media")
		return nil
	}

	args := make([]string, 2)
	args[0] = "-i"
	args[1] = e.Path
	args = append(args, e.ffmpegArgs()...)
	args = append(args, []string{"-y", e.tmpfile}...)

	log.WithFields(log.Fields{
		"src_path":    e.Path,
		"tmp_path":    e.tmpfile,
		"ffmpeg_args": args,
	}).Info("media: transcoding file")

	cmd := exec.Command("ffmpeg", args...)

	start := time.Now()
	e.Stats.CommandStdout, err = cmd.CombinedOutput()
	e.Stats.Duration = time.Now().Sub(start)
	e.Stats.EndSizeBytes = e.getSizeBytes(e.tmpfile)

	if err != nil {
		e.abort()
		return fmt.Errorf("media: error during transcoding: %s", err)
	}

	if err := e.rename(); err != nil {
		e.abort()
		return fmt.Errorf("media: error renaming file at %s: %s", e.Path, err)
	}

	if !viper.GetBool("transcoding.keep_originals") {
		if err := e.cleanup(); err != nil {
			log.WithField("error", err).Errorf("media: error cleaning up unneeded files")
		}
	}

	e.TranscodeSuccess = true

	log.WithFields(log.Fields{
		"path":       e.Path,
		"duration":   e.Stats.Duration,
		"start_size": e.Stats.InitialSizeBytes,
		"end_size":   e.Stats.EndSizeBytes,
	}).Info("media: transcode complete")

	return nil
}
