package renamer

import (
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Rule struct {
	Old string `yaml:"old"`
	New string `yaml:"new"`

	oldMatcher *regexp.Regexp
}

// Applies this rename rule if the original path matches
func (r *Rule) Apply(path string) string {
	if err := r.compileMatcher(); err != nil {
		log.WithError(err).WithField("regexp", r.Old).Error("failed to compile rename regexp")
		return path
	}

	if !filepath.IsAbs(path) {
		log.WithField("path", path).Error("path is not absolute")
		return path
	}

	// only apply the rules to the directory and file, not all of the path
	elems := strings.Split(path, string(filepath.Separator))
	prefixLen := len(elems) - 2
	if prefixLen < 0 {
		prefixLen = 0
	}
	target := filepath.Join(elems[prefixLen:]...)

	if r.oldMatcher.MatchString(target) {
		log.WithFields(log.Fields{
			"regexp":  r.Old,
			"replace": r.New,
		}).Info("applying rename rule")
		target = r.oldMatcher.ReplaceAllString(target, r.New)
	}

	prefix := filepath.Join(elems[:prefixLen]...)
	return string(filepath.Separator) + filepath.Join(prefix, target)
}

func (r *Rule) compileMatcher() error {
	var err error
	if r.oldMatcher != nil {
		return nil
	}

	r.oldMatcher, err = regexp.Compile("(?i)" + r.Old)
	if err != nil {
		return err
	}
	return nil
}

func LoadRules() ([]Rule, error) {
	var rules []Rule

	err := viper.UnmarshalKey("rename.rules", &rules)
	if err != nil {
		return rules, err
	}

	return rules, nil
}
