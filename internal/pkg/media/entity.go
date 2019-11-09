package media

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/vansante/go-ffprobe"
)

type Stats struct {
	Duration         time.Duration `json:"duration"`
	InitialSizeBytes uint64        `json:"initial_size_bytes"`
	EndSizeBytes     uint64        `json:"end_size_bytes"`
	CommandStdout    []byte        `json:"command_stdout"`
}

type Config struct {
	OnlySD    bool
	AudioArgs []string
	VideoArgs []string
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
	Media            Type  `json:"type"`
	Stats            Stats `json:"stats"`
	TranscodeSuccess bool  `json:"transcode_success"`

	skipTranscode bool
	config        Config
	basename      string
}

func NewEntity(details Details, mediaConfig Config) (*Entity, error) {
	if details.Path == "" {
		return nil, fmt.Errorf("media: path must not be empty")
	}

	return &Entity{
		Details: details,
		Stats:   Stats{},
		config:  mediaConfig,
	}, nil
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
		if e.config.OnlySD {
			log.Info("media: skipping transcode, only_sd is set")
			e.skipTranscode = true
		}
	default:
		e.Media = MEDIA_VIDEO
	}
	return nil
}

func (e *Entity) tempFilename() string {
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

	tmpPath := filepath.Join(dir, uuid.New().String()+ext)
	log.WithField("path", tmpPath).Debug("media: temporary path for encoding media")
	return tmpPath
}

func (e *Entity) ffmpegArgs() []string {
	if e.Media == MEDIA_VIDEO || e.Media == MEDIA_H264_VIDEO {
		return e.config.VideoArgs
	}
	return e.config.AudioArgs
}

func (e *Entity) replace(src string) error {
	dest := e.Path
	if e.Media == MEDIA_AUDIO {
		dest = strings.Replace(dest, filepath.Ext(e.Path), ".mp3", -1)
	}

	log.WithFields(log.Fields{
		"src":  src,
		"dest": dest,
	}).Info("media: renaming transcoded file")
	return os.Rename(src, dest)
}

func (e *Entity) cleanup() error {
	if e.Media != MEDIA_AUDIO {
		return nil
	}

	log.WithField("path", e.Path).Info("media: removing original file")
	if err := os.Remove(e.Path); err != nil {
		return err
	}

	// Overwrite file path with the corrected path for an MP3
	e.Path = strings.Replace(e.Path, filepath.Ext(e.Path), ".mp3", -1)
	return nil
}

func (e *Entity) abort() error {
	return os.Remove(e.tempFilename())
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

func (e *Entity) DestPath() string {
	return e.tempFilename()
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

	tmpfile := e.tempFilename()

	args := make([]string, 2)
	args[0] = "-i"
	args[1] = e.Path
	args = append(args, e.ffmpegArgs()...)
	args = append(args, []string{"-y", tmpfile}...)

	log.WithFields(log.Fields{
		"src_path":    e.Path,
		"tmp_path":    tmpfile,
		"ffmpeg_args": args,
	}).Info("media: transcoding file")

	cmd := exec.Command("ffmpeg", args...)

	start := time.Now()
	e.Stats.CommandStdout, err = cmd.CombinedOutput()
	e.Stats.Duration = time.Now().Sub(start)
	e.Stats.EndSizeBytes = e.getSizeBytes(tmpfile)

	if err != nil {
		e.abort()
		return fmt.Errorf("media: error during transcoding: %s", err)
	}

	if err := e.replace(tmpfile); err != nil {
		e.abort()
		return fmt.Errorf("media: error replacing file at %s: %s", e.Path, err)
	}

	if err := e.cleanup(); err != nil {
		log.WithField("error", err).Errorf("media: error cleaning up unneeded files")
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
