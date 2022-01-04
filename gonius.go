package gonius

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
)

var ErrNotFound = errors.New("no results found")

// TODO: create a single regex
var (
	stageOne   = regexp.MustCompile(` *\([^)]*\) *`)
	stageTwo   = regexp.MustCompile(` *\[[^\]]*]`)
	stageThree = regexp.MustCompile(`feat.|ft.`)
)

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
	data, err := makeRequest(fmt.Sprintf("https://genius.com/api/search/multi?per_page=5&q=%s", formatQuery(query)))
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

func (s *Song) Lyrics() (string, error) {
	rawData, err := makeRequest(s.URL)
	if err != nil {
		return "", err
	}

	data := string(rawData)

	firstIndex := strings.Index(data, "JSON.parse('")
	if firstIndex == -1 {
		return "", errors.New("could not find JSON inside html")
	}

	lastIndex := strings.Index(data[firstIndex:], "');")
	if lastIndex == -1 {
		return "", errors.New("could not find JSON inside html")
	}

	data = strings.ReplaceAll(data[firstIndex+12:firstIndex+lastIndex], `\"`, `"`)
	data = strings.ReplaceAll(strings.ReplaceAll(data, `\'`, `'`), `\\`, `\`)

	return extractLyrics(gjson.Get(data, "songPage.lyricsData.body.children")), nil
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

	return ioutil.ReadAll(res.Body)
}

func formatQuery(query string) string {
	query = stageOne.ReplaceAllString(stageTwo.ReplaceAllString(query, ""), "")
	return url.QueryEscape(stageThree.ReplaceAllString(strings.ToLower(query), ""))
}
