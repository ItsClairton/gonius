package gonius

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
)

var ErrNotFound = errors.New("no results found")
var responsePattern = regexp.MustCompile(`window\.__PRELOADED_STATE__ = JSON\.parse\('(.*?)'\);`)

type Song struct {
	ID  int    `json:"id"`
	URL string `json:"url"`

	Title     string `json:"title"`
	FullTitle string `json:"full_title"`

	Image     string `json:"song_art_image_url"`
	Thumbnail string `json:"song_art_image_thumbnail_url"`

	PrimaryArtist struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Image string `json:"image_url"`
	} `json:"primary_artist"`
}

func SearchSong(query string) ([]*Song, error) {
	data, err := makeRequest(fmt.Sprintf("https://genius.com/api/search/multi?per_page=5&q=%s", url.QueryEscape(query)))
	if err != nil {
		return nil, err
	}

	mapper, results := map[int]bool{}, []*Song{}
	gjson.GetBytes(data, `response.sections.#.hits.#(type=="song").result`).ForEach(func(_, value gjson.Result) bool {
		var entry *Song

		if err := json.Unmarshal([]byte(value.String()), &entry); err == nil && !mapper[entry.ID] {
			mapper[entry.ID], results = true, append(results, entry)
		}

		return true
	})

	if len(results) == 0 {
		return nil, ErrNotFound
	}

	return results, nil
}

func GetLyrics(songURL string) (string, error) {
	rawData, err := makeRequest(songURL)
	if err != nil {
		if err.Error() == "the server responded with unexpected 404 Not Found status" {
			return "", ErrNotFound
		}

		return "", err
	}

	data := string(rawData)

	matches := responsePattern.FindStringSubmatch(data)
	if len(matches) < 2 {
		return "", errors.New("could not extract JSON from HTML")
	}

	data = strings.ReplaceAll(matches[1], `\"`, `"`)
	data = strings.ReplaceAll(strings.ReplaceAll(data, `\'`, `'`), `\\`, `\`)

	return extractLyrics(gjson.Get(data, "songPage.lyricsData.body.children")), nil
}

func (s *Song) Lyrics() (string, error) {
	return GetLyrics(s.URL)
}

func extractLyrics(data gjson.Result) (lyrics string) {
	data.ForEach(func(_, value gjson.Result) bool {
		switch value.Type {
		case gjson.String:
			lyrics += value.String()
		case gjson.JSON:
			if tag := value.Get("tag").String(); tag == "br" || tag == "inread-ad" {
				lyrics += "\n"
			} else if value = value.Get("children"); value.Exists() {
				lyrics += extractLyrics(value)
			}
		}

		return true
	})

	return lyrics
}

func makeRequest(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("the server responded with unexpected %s status", res.Status)
	}

	return io.ReadAll(res.Body)
}
