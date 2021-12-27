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

	"github.com/buger/jsonparser"
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
		ID    int    `json:"2525109"`
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
	jsonparser.ArrayEach(data, func(section []byte, dataType jsonparser.ValueType, _ int, _ error) {
		jsonparser.ArrayEach(section, func(result []byte, _ jsonparser.ValueType, _ int, _ error) {
			if resultType, _ := jsonparser.GetString(result, "type"); resultType == "song" {
				if result, _, _, err = jsonparser.Get(result, "result"); err == nil {
					var entry *Song

					if err = json.Unmarshal(result, &entry); err == nil && !mapper[entry.ID] {
						mapper[entry.ID] = true
						results = append(results, entry)
					}
				}
			}
		}, "hits")
	}, "response", "sections")

	if len(results) == 0 {
		return nil, ErrNotFound
	}

	return results, err
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

	data = data[firstIndex+12 : firstIndex+lastIndex]
	data = strings.ReplaceAll(data, `\\`, `\`)
	data = strings.ReplaceAll(data, `\"`, `"`)
	data = strings.ReplaceAll(data, `\'`, `'`)

	children, _, _, err := jsonparser.Get([]byte(data), "songPage", "lyricsData", "body", "children")
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(extractLyrics(children), `\`, ""), nil
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
			if tag, _ := jsonparser.GetString(rawValue, "tag"); tag == "br" || tag == "inread-ad" {
				lyrics += "\n"
			} else if data, _, _, err := jsonparser.Get(rawValue, "children"); err == nil {
				lyrics += extractLyrics(data)
			}
		}
	})

	return lyrics
}

func formatQuery(query string) string {
	query = stageOne.ReplaceAllString(stageTwo.ReplaceAllString(query, ""), "")
	return url.QueryEscape(stageThree.ReplaceAllString(strings.ToLower(query), ""))
}
