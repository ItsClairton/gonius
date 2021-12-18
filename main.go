package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/buger/jsonparser"
)

var ErrNotFound = errors.New("no results found")

type Song struct {
	ID  int    `json:"id"`
	URL string `json:"url"`

	Title     string `json:"title"`
	FullTitle string `json:"full_title"`

	Image     string `json:"song_art_image_url"`
	Thumbnail string `json:"song_art_image_thumbnail_url"`

	Artist string `json:"artist_names"`
}

func SearchSong(query string) (*Song, error) {
	data, err := makeRequest(fmt.Sprintf("https://genius.com/api/search/multi?per_page=1&q=%s", url.QueryEscape(query)))
	if err != nil {
		return nil, err
	}

	data, _, _, err = jsonparser.Get(data, "response", "sections", "[1]", "hits", "[0]", "result")
	if err != nil {
		if err == jsonparser.KeyPathNotFoundError {
			return nil, ErrNotFound
		}

		return nil, err
	}

	var result *Song
	err = json.Unmarshal(data, &result)

	return result, err
}

func (s *Song) Lyrics() (string, error) {
	rawData, err := makeRequest(s.URL)
	if err != nil {
		return "", err
	}

	data := strings.ReplaceAll(string(rawData), "\\", "")

	firstIndex := strings.Index(data, "JSON.parse('") + 12
	lastIndex := strings.Index(data[firstIndex:], "');")
	if firstIndex == 12 || lastIndex == -1 {
		return "", errors.New("could not find JSON inside html")
	}

	data = data[firstIndex : firstIndex+lastIndex]
	children, _, _, err := jsonparser.Get([]byte(data), "songPage", "lyricsData", "body", "children")
	if err != nil {
		return "", err
	}

	return extractLyrics(children), nil
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

	return ioutil.ReadAll(res.Body)
}

func extractLyrics(data []byte) (lyrics string) {
	jsonparser.ArrayEach(data, func(rawValue []byte, dataType jsonparser.ValueType, _ int, _ error) {
		switch dataType {
		case jsonparser.String:
			lyrics += string(rawValue)
		case jsonparser.Object:
			if tag, _ := jsonparser.GetString(rawValue, "tag"); tag == "br" {
				lyrics += "\n"
			} else if data, _, _, err := jsonparser.Get(rawValue, "children"); err == nil {
				lyrics += extractLyrics(data)
			}
		}
	})

	return lyrics
}
