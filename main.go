package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"text/template"
)

const (
	numWantedStories = 3

	topStoriesURL = "https://hacker-news.firebaseio.com/v0/topstories.json"
	storyURL      = "https://hacker-news.firebaseio.com/v0/item/"
)

type News struct {
	topStoryIDs []int
	Headlines   []string
}

type Story struct {
	By    string `json:"by"`
	Title string `json:"title"`
	Type  string `json:"type"`
	Url   string `json:"url"`
}

func (n *News) fetch() {
	n.loadTopStoryIDs()
	n.loadHeadlines()
}

// loadTopStoryIDs loads the top 500 story ids.
func (n *News) loadTopStoryIDs() {
	resp, err := http.Get(topStoriesURL)
	if err != nil {
		log.Fatalf("problem loading top news stories: %v\n", err)
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&n.topStoryIDs); err != nil {
		log.Fatalf("error parsing top stories ids: %v\n", err)
	}
}

func (n *News) loadHeadlines() {
	n.Headlines = []string{}
	var story Story

	for i := 0; i < numWantedStories; i++ {
		story = fetchStory(n.topStoryIDs[i])
		n.Headlines = append(n.Headlines, story.Title)
	}
}

func fetchStory(id int) (story Story) {
	url := storyURL + strconv.Itoa(id) + ".json"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("problem loading story #%d: %v\n", id, err)
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&story); err != nil {
		log.Fatalf("error parsing story: %v\n", err)
	}
	return
}

func newsHandler(w http.ResponseWriter, r *http.Request) {
	t := template.New("hackernews")
	t = template.Must(template.ParseFiles("templates/news.html"))

	news := News{}
	news.fetch()

	t.Execute(w, news)
}

func main() {
	http.HandleFunc("/", newsHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
