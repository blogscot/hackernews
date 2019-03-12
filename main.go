package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"text/template"
)

const (
	numWantedStories = 30

	topStoriesURL = "https://hacker-news.firebaseio.com/v0/topstories.json"
	storyURL      = "https://hacker-news.firebaseio.com/v0/item/"
)

type News struct {
	topStoryIDs []int
	Stories     map[int]Story
}

type Story struct {
	ID    int    `json:"id"`
	By    string `json:"by"`
	Title string `json:"title"`
	Type  string `json:"type"`
	Url   string `json:"url"`
}

func (n *News) fetch() {
	n.loadTopStoryIDs()
	n.loadStories()
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

func (n *News) loadStories() {
	var wg sync.WaitGroup
	n.Stories = map[int]Story{}
	storyChan := make(chan Story)

	for i := 0; i < numWantedStories; i++ {
		wg.Add(1)
		go fetchStory(n.topStoryIDs[i], storyChan, &wg)
	}

	go func() {
		wg.Wait()
		close(storyChan)
	}()

	for story := range storyChan {
		n.Stories[story.ID] = story
	}
}

func fetchStory(id int, ch chan Story, wg *sync.WaitGroup) {
	defer wg.Done()
	var story Story
	url := storyURL + strconv.Itoa(id) + ".json"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("problem loading story #%d: %v\n", id, err)
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&story); err != nil {
		log.Fatalf("error parsing story: %v\n", err)
	}
	ch <- story
}

func newsHandler(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFiles("templates/news.html"))

	news := News{}
	stories := make([]Story, numWantedStories)

	news.fetch()

	// order stories according to topStoryIDs
	for index, id := range news.topStoryIDs[:numWantedStories] {
		stories[index] = news.Stories[id]
	}

	_ = t.Execute(w, stories)
}

func main() {
	http.HandleFunc("/", newsHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
