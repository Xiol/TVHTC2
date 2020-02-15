package renamer

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var timestampMatcher *regexp.Regexp
var fullTimestampMatcher *regexp.Regexp
var newMatcher *regexp.Regexp
var whitespaceCleaner *regexp.Regexp

func init() {
	timestampMatcher = regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})(\d{2})-(\d{2})`)
	newMatcher = regexp.MustCompile(`New_?-?`)
	fullTimestampMatcher = regexp.MustCompile(`\d{4}-\d{2}-\d{2,4}T?\d{2}?-?(?:\d{4}|\d{2})`)
	whitespaceCleaner = regexp.MustCompile(`\s+`)
}

// A Renamer is used to alter a given path to neaten programme names and timestamps
type Renamer struct {
	FixTimestamps bool
	FixSpacing    bool
	RemoveNew     bool
}

// NewRenamer returns a Renamer that will fix timestamp formatting, remove 'New' from titles
// and fix spacing.
func NewRenamer() Renamer {
	// We initailise the renamer from the config at this point to avoid configuration
	// file changes altering the control flow mid-rename. A new instance of Renamer should
	// be used for every entity.
	return Renamer{
		FixTimestamps: viper.GetBool("rename.fix_timestamps"),
		RemoveNew:     viper.GetBool("rename.remove_new"),
		FixSpacing:    viper.GetBool("rename.fix_spacing"),
	}
}

// Rename renames the provided path based on the Renamer settings
func (r *Renamer) Rename(path string) string {
	path = r.fixTimestamps(path)
	path = r.removeNew(path)
	path = r.fixSpacing(path)
	path = r.applyRules(path)
	return path
}

func (r *Renamer) fixTimestamps(path string) string {
	if !r.FixTimestamps {
		return path
	}

	matches := timestampMatcher.FindStringSubmatch(path)
	if matches == nil {
		// no timestamp
		return path
	}

	formattedDate := fmt.Sprintf("%s-%s-%sT%s%s", matches[1], matches[2], matches[3], matches[4], matches[5])
	return timestampMatcher.ReplaceAllString(path, formattedDate)
}

func (r *Renamer) removeNew(path string) string {
	if !r.RemoveNew {
		return path
	}

	// Only check the file so we can anchor the regex properly
	_, file := filepath.Split(path)
	match := newMatcher.FindStringIndex(file)
	if match == nil || match[0] != 0 {
		// No 'New' or it's part of the programme name
		return path
	}

	// We need to remove New from both the directory name
	// and the file name. It's up to the caller to ensure
	// the directory exists for moving the file into.
	return newMatcher.ReplaceAllString(path, "")
}

func (r *Renamer) fixSpacing(path string) string {
	if !r.FixSpacing {
		return path
	}

	// Remove timestamp from the end of the path first so we don't
	// adjust it during space fixing.
	timestamp := r.getTimestamp(path)
	path = strings.Replace(path, timestamp, "", -1)

	path = strings.Replace(path, "_/", "/", -1)
	path = strings.Replace(path, "-/", "/", -1)
	path = strings.Replace(path, "_.", ".", -1)
	path = strings.Replace(path, "-", " ", -1)
	path = strings.Replace(path, "_", " - ", -1)
	path = strings.Replace(path, "...", "", -1)

	// reinsert the timestamp with fixed spacing
	ext := filepath.Ext(path)
	path = strings.Replace(path, ext, "", -1)
	path = path + " - " + timestamp + ext

	// remove extra whitespace
	path = whitespaceCleaner.ReplaceAllString(path, " ")

	return path
}

func (r *Renamer) getTimestamp(path string) string {
	matches := fullTimestampMatcher.FindStringSubmatch(path)
	if matches != nil {
		return matches[0]
	}
	return ""
}

func (r *Renamer) applyRules(path string) string {
	rules, err := LoadRules()
	if err != nil {
		log.WithError(err).Error("renamer: error loading rename rules")
		return path
	}

	for i := range rules {
		path = rules[i].Apply(path)
	}

	return path
}
