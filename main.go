package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"text/template"
	"time"
)

const (
	numWantedStories = 30
	refreshTimer     = 15 * time.Minute

	topStoriesURL  = "https://hacker-news.firebaseio.com/v0/topstories.json"
	storyURL       = "https://hacker-news.firebaseio.com/v0/item/"
	ycombinatorURL = "https://news.ycombinator.com/item?id="
)

type News struct {
	topStoryIDs []int
	Stories     map[int]Story
}

type Story struct {
	By    string `json:"by"`
	ID    int    `json:"id"`
	Score int    `json:"score"`
	Time  int    `json:"time"`
	Title string `json:"title"`
	Type  string `json:"type"`
	Url   string `json:"url"`
}

var (
	storiesCached = false
	mutex         = &sync.RWMutex{}

	newsInstance *News
	once         sync.Once
)

// getNewsInstance returns a singleton news item.
func getNewsInstance() *News {
	once.Do(func() {
		if newsInstance == nil {
			newsInstance = &News{}
		}
	})
	return newsInstance
}

// fetch loads and caches the top stories IDs and story headlines.
// While the cache is being updated callers are paused until this
// activity has completed.
func (n *News) fetch() {
	mutex.Lock()
	defer mutex.Unlock()

	if !storiesCached {
		storiesCached = true
		log.Println("loading stories...")
		n.loadTopStoryIDs()
		n.loadStories()

		time.AfterFunc(refreshTimer, func() {
			mutex.Lock()
			storiesCached = false
			mutex.Unlock()
			n.fetch()
		})
	}
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

// sortStories orders stories according to the Top Stories ordering.
func (n *News) sortStories() (stories []Story) {
	stories = make([]Story, numWantedStories)

	mutex.RLock()
	for index, id := range n.topStoryIDs[:numWantedStories] {
		stories[index] = n.Stories[id]
	}
	mutex.RUnlock()
	return
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

	if story.Url == "" {
		story.Url = ycombinatorURL + strconv.Itoa(story.ID)
	}

	ch <- story
}

func newsHandler(w http.ResponseWriter, r *http.Request) {
	funcs := template.FuncMap{
		"hostname": func(raw string) string {
			u, _ := url.Parse(raw)
			return fmt.Sprintf("(%s)", u.Hostname())
		},
	}
	t := template.Must(template.New("news.html").Funcs(funcs).ParseFiles("templates/news.html"))

	news := getNewsInstance()
	news.fetch()

	err := t.Execute(w, news.sortStories())
	if err != nil {
		fmt.Printf("error executing template: %v\n", err)
	}
}

func main() {
	http.HandleFunc("/", newsHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
