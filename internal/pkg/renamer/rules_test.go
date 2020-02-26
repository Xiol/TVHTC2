package renamer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuleApply(t *testing.T) {
	r := Rule{
		Old: "8 Out of 10 Cats Does",
		New: "8 Out of 10 Cats Does Countdown",
	}

	assert.Equal(t, "/srv/storage/media/DVR/8 Out of 10 Cats Does Countdown/8 Out of 10 Cats Does Countdown - 2020-01-01T2100.mkv",
		r.Apply("/srv/storage/media/DVR/8 Out of 10 Cats Does/8 Out of 10 Cats Does - 2020-01-01T2100.mkv"))
	assert.Equal(t, "/srv/storage/media/DVR/8 Out of 10 Cats Does Countdown/8 Out of 10 Cats Does Countdown - 2020-01-01T2100.mkv",
		r.Apply("/srv/storage/media/DVR/8 out of 10 cats does/8 out OF 10 cAtS DOES - 2020-01-01T2100.mkv"))
	assert.Equal(t, "/srv/storage/media/DVR/Eastenders/Eastenders - 2020-01-01T2100.mkv",
		r.Apply("/srv/storage/media/DVR/Eastenders/Eastenders - 2020-01-01T2100.mkv"))

	r = Rule{
		Old: "media",
		New: "foobar",
	}
	assert.Equal(t, "/srv/storage/media/foobar/foobar", r.Apply("/srv/storage/media/media/media"))

	r = Rule{
		Old: "Live - The Last Leg",
		New: "The Last Leg",
	}
	assert.Equal(t, "/srv/storage/media/DVR/The Last Leg/The Last Leg - 2020-02-21T2200.mkv",
		r.Apply("/srv/storage/media/DVR/Live - The Last Leg/Live - The Last Leg - 2020-02-21T2200.mkv"))
}
