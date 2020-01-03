package renamer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRenamer_fixTimestamps(t *testing.T) {
	r := NewRenamer()

	orig := "/srv/storage/dvr/Dracula/Dracula2020-01-0121-00.mkv"
	expected := "/srv/storage/dvr/Dracula/Dracula2020-01-01T2100.mkv"

	assert.Equal(t, expected, r.fixTimestamps(orig))
}

func TestRenamer_removeNew(t *testing.T) {
	r := NewRenamer()

	orig := "/srv/storage/dvr/New_-Bancroft/New_-Bancroft2020-01-0121-00.mkv"
	expected := "/srv/storage/dvr/Bancroft/Bancroft2020-01-0121-00.mkv"

	assert.Equal(t, expected, r.removeNew(orig))
}

func TestRenamer_fixSpacing(t *testing.T) {
	r := NewRenamer()
	r.FixTimestamps = false

	tests := [][]string{
		[]string{
			"/srv/storage/dvr/Wallace-&-Gromit_-A-Close-Shave/Wallace-&-Gromit_-A-Close-Shave2020-01-0121-00.mkv",
			"/srv/storage/dvr/Wallace & Gromit - A Close Shave/Wallace & Gromit - A Close Shave - 2020-01-0121-00.mkv",
		},
		[]string{
			"/srv/storage/dvr/Wallace-&-Gromit_-The-Wrong.../Wallace-&-Gromit_-The-Wrong...2019-12-2416-35.mkv",
			"/srv/storage/dvr/Wallace & Gromit - The Wrong/Wallace & Gromit - The Wrong - 2019-12-2416-35.mkv",
		},
		[]string{
			"/srv/storage/dvr/New_-Richard-Osman's-World-Cup.../New_-Richard-Osman's-World-Cup...2019-12-2721-00.mkv",
			"/srv/storage/dvr/New - Richard Osman's World Cup/New - Richard Osman's World Cup - 2019-12-2721-00.mkv",
		},
		[]string{
			"/srv/storage/dvr/Dragons'-Den_-Pitches-to-Riches_/Dragons'-Den_-Pitches-to-Riches_2019-12-24T2000.mkv",
			"/srv/storage/dvr/Dragons' Den - Pitches to Riches/Dragons' Den - Pitches to Riches - 2019-12-24T2000.mkv",
		},
		[]string{
			"/srv/storage/dvr/James-Martin's-Home-Comforts-/James-Martin's-Home-Comforts2020-01-0121-30.mkv",
			"/srv/storage/dvr/James Martin's Home Comforts/James Martin's Home Comforts - 2020-01-0121-30.mkv",
		},
	}

	for _, test := range tests {
		assert.Equal(t, test[1], r.fixSpacing(test[0]))
	}
}
