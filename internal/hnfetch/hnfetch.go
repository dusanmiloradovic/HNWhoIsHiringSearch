package hnfetch

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const searchApi = "https://hn.algolia.com/api/v1/search_by_date?query=%22who%20is%20hiring%22&tags=story"
const itemApi = "https://hacker-news.firebaseio.com/v0/item/%s.json"

type HNWhosHiring struct {
	Title                string // like May 2024
	PostId               string
	childPostIds         []string
	lastTimestampChecked int64
}

type HHWhoIsHiringWithError struct {
	WhoIsHiring *HNWhosHiring
	Error       error
}

type HNWhoIsHiringPost struct {
	PostId      int
	Title       *string // includes as per the HN recommendation location, company, job title . If this can't  be parsed, this will be used by client instead
	Location    *string
	Company     *string
	JobTitle    *string
	Salary      *string // try to deduct
	Posted      int
	Description string
	Remote      bool
}

var whoisHiringChan = make(chan *HHWhoIsHiringWithError)
var jobMap = make(map[string]*HNWhoIsHiringPost)

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

type HNPostResponse struct {
	By   string `json:"by"`
	Id   int    `json:"id"`
	Text string `json:"text"`
	Time int64  `json:"time"`
}

type HNFetch interface {
	Init() error
	GetPosts(cursor *string, fetchSize int)
	LastWhoIsHiring() (*HNWhosHiring, error) // remoeve
}

type HNAPI struct {
	existingWhoIsHiring *HNWhosHiring
}

func (s HNAPI) findCursorIndex(cursor *string) int {
	if cursor != nil {
		return -1 //start from the beginning
	}
	for i := 0; i < len(s.existingWhoIsHiring.childPostIds); i++ {
		if *cursor == s.existingWhoIsHiring.childPostIds[i] {
			return i
		}
	}
	return -1 // cursor not found, start from the beginning
}

func (s HNAPI) GetPosts(cursor *string, fetchSize int) []HNWhoIsHiringPost {
	if s.existingWhoIsHiring == nil {
		s.LastWhoIsHiring()
	}
	posts := make(chan *HNWhoIsHiringPost)
	cursorInd := s.findCursorIndex(cursor)
	for i := cursorInd + 1; i < len(s.existingWhoIsHiring.childPostIds); i++ {
		if i == (fetchSize - cursorInd - 1) {
			break
		}
		go s.getPost(posts, i)
	}
	l := len(s.existingWhoIsHiring.childPostIds) - cursorInd - 1
	if l > fetchSize {
		l = fetchSize
	}
	var ret []HNWhoIsHiringPost
	for i := 0; i < l; i++ {
		ret = append(ret, *(<-posts))
	}
	return ret
}

func (s HNAPI) getPost(posts chan *HNWhoIsHiringPost, i int) {
	childPosts := s.existingWhoIsHiring.childPostIds
	job := jobMap[childPosts[i]]
	if job == nil {
		url := fmt.Sprintf(itemApi, childPosts[i])
		resp, err := http.Get(url)
		if err != nil {
			slog.Error(fmt.Sprintf("Error when reading %s", url), err)
			posts <- nil
			return
		}
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			posts <- nil
			slog.Error(fmt.Sprintf("Error when reading %s body", url), err)
			return
		}
		var response HNPostResponse
		err = json.Unmarshal(bodyBytes, &response)
		if err != nil {
			slog.Error(fmt.Sprintf("Error when parsing %s body", url), err)
			posts <- nil
			return
		}
		parser := NewPostMeta(response.Text)
		post := HNWhoIsHiringPost{
			PostId:  response.Id,
			Title:   parser.GetJobTitle(),
			Company: parser.GetCompany(),
			Remote:  parser.IsRemote(),
		}
		jobMap[childPosts[i]] = &post
		posts <- &post
	} else {
		slog.Debug("Cache hit")
		posts <- job
	}
}

func (s HNAPI) findLastWhosHiringPost() (*HNWhosHiring, error) {

	resp, err := http.DefaultClient.Get(searchApi)
	if err != nil {
		chErr := &HHWhoIsHiringWithError{}
		chErr.Error = err
		whoisHiringChan <- chErr
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
	chVar := &HHWhoIsHiringWithError{ret, nil}
	whoisHiringChan <- chVar
	return ret, nil
}

func (s HNAPI) LastWhoIsHiring() (*HNWhosHiring, error) {
	if s.existingWhoIsHiring != nil && s.existingWhoIsHiring.lastTimestampChecked-time.Now().UnixMilli() < 60000 {
		return s.existingWhoIsHiring, nil
	}
	go s.findLastWhosHiringPost()
	ret := <-whoisHiringChan
	return ret.WhoIsHiring, ret.Error
}

func (s HNAPI) Init() error {
	post, err := s.LastWhoIsHiring()
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
