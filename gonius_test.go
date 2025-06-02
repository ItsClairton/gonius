package gonius

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SearchSong(t *testing.T) {
	results, err := SearchSong("shelter - madeon")
	assert.NoError(t, err)
	assert.NotEmpty(t, results)

	for _, song := range results {
		assert.NotEmpty(t, song.ID)
		assert.NotEmpty(t, song.URL)
		assert.NotEmpty(t, song.Title)
		assert.NotEmpty(t, song.FullTitle)
		assert.NotEmpty(t, song.Image)
		assert.NotEmpty(t, song.Thumbnail)
		assert.NotEmpty(t, song.PrimaryArtist.ID)
		assert.NotEmpty(t, song.PrimaryArtist.Name)
		assert.NotEmpty(t, song.PrimaryArtist.Image)
	}
}

func Test_SearchSong_NotFound(t *testing.T) {
	results, err := SearchSong("")
	assert.Equal(t, ErrNotFound, err)
	assert.Empty(t, results)
}

func Test_GetLyrics(t *testing.T) {
	lyrics, err := GetLyrics("https://genius.com/Porter-robinson-and-madeon-shelter-lyrics")
	assert.NoError(t, err)
	assert.NotEmpty(t, lyrics)

	println(lyrics)
}

func Test_GetLyrics_NotFound(t *testing.T) {
	lyrics, err := GetLyrics("https://genius.com/Nonexistent-song-lyrics")
	assert.Equal(t, ErrNotFound, err)
	assert.Empty(t, lyrics)
}
