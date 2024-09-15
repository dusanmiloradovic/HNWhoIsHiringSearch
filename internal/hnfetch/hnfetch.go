package hnfetch

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

const searchApi = "https://hn.algolia.com/api/v1/search_by_date?query=%22who%20is%20hiring%22&tags=story"
const itemApi = "https://hacker-news.firebaseio.com/v0/item/40420850.json"

type HNWhosHiring struct {
	Title                string // like May 2024
	PostId               string
	childPostIds         []string
	lastTimestampChecked int64
}

type SearchAPIResponse struct {
	Hits []struct {
		Author          string `json:"author"`
		HighlightResult struct {
			Title struct {
				Value string `json:"value"`
			} `json:"title"`
		} `json:"_highlightResult"`
		Children []string `json:"children"`
		StoryId  string   `json:"story_id"`
	} `json:"hits"`
}

type HNFetch interface {
	Init() error
	GetPosts(cursor int, fetchSize int)      //instead of optional parameters, 0 means there is no cursor
	LastWhoIsHiring() (*HNWhosHiring, error) // remoeve
}

type HNAPI struct {
	existingWhoIsHiring *HNWhosHiring
}

func (s HNAPI) GetPosts(cursor int, fetchSize int) {
	//TODO implement me
	panic("implement me")
}

func (s HNAPI) findLastWhosHiringPost() (*HNWhosHiring, error) {
	resp, err := http.DefaultClient.Get(searchApi)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)

	var response SearchAPIResponse

	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		return nil, err
	}
	ret := &HNWhosHiring{}
	for j := 0; j < len(response.Hits); j++ {
		if response.Hits[j].Author == "whoishiring" {
			ret.childPostIds = response.Hits[j].Children
			ret.lastTimestampChecked = time.Now().UnixMilli()
			ret.PostId = response.Hits[j].StoryId
			// by definition its ordered, take the first matching
			break
		}
	}
	return ret, nil
}

func (s HNAPI) Init() error {
	if s.existingWhoIsHiring != nil && s.existingWhoIsHiring.lastTimestampChecked-time.Now().UnixMilli() < 60000 {
		return nil

	}
	post, err := s.findLastWhosHiringPost()
	if err != nil {
		return err
	}
	s.existingWhoIsHiring = post
	return nil
}

func NewHNFetch() HNFetch {
	ret := HNAPI{
		existingWhoIsHiring: nil,
	}
	return ret
}
