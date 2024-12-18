package main

import (
	"net/http"
	"net/url"
	"log"
	"io/ioutil"
	"fmt"
	"encoding/json"
	"time"
	"4d63.com/tz"
	"sort"
	"strings"
	"bytes"
	"text/template"
	"os"
	"strconv"
	"fyne.io/fyne/widget"
	"fyne.io/fyne/app"
	"fyne.io/fyne"
)

var apiKey string = # API key Get from Admins
var forumList string = "10,11,12,13,14,15,16,17,18,36,37,38,29,30,31,32,33,34,47,46,44,40,24,25,28,54,42,64,72,73,65,66,67,56,68,69,70,57,49,50"

type Member struct {
	Id int `json:"id"`
	Name string `json:"name"`
	Title string `json:"title"`
	Timezone string `json:"timezone"`
	FormattedName string `json:"formattedName"`
}

type Post struct {
	Id int `json:"id"`
	Item_id int `json:"item_id"`
	Author Member `json:"author"`
	Date time.Time `json:"date"`
	Content string `json:"content"`
	Url string `json:"url"`
}

type Topic struct {
	Id int `json:"id"`
	Title string `json:"title"`
	Prefix string `json:"prefix"`
	Tags []string `json:"tags"`
	FirstPost Post `json:"firstPost"`
	LastPost Post `json:"lastPost"`
	Url string `json:"url"`
}

type TopicCount struct {
	Id int
	Title string
	Count int
	Url string
}

type UserCount struct {
	Id int
	Name string
	Topics []TopicCount
}


func basicAuth(path string) []byte {
	var url string = "http://www.freedomplaybypost.com/api/" + path
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(apiKey, "")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	json_body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	return json_body
}

func BeginningOfMonth(t time.Time) time.Time {
	EDT, err := tz.LoadLocation("America/New_York")
		if err != nil {
			log.Fatal(err)
		}
    return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, EDT).UTC()
}

func EndOfMonth(t time.Time) time.Time {
    return BeginningOfMonth(t).AddDate(0, 1, 0).Add(-time.Second)
}

func GetValidTopics(begin time.Time, end time.Time, progress *widget.ProgressBar) map[int]string {
	type Response struct {
		Page int
		PerPage int
		TotalResults int
		TotalPages int
		Results []Topic `json:"results"`
	}
	valid := make(map[int]string)
	var getNext bool = true
	var page int = 0
	progress.Show()
	for getNext {
		page += 1
		path := fmt.Sprintf("forums/topics/?sortBy=date&sortDir=desc&archived=0&hidden=0&page=%d&perPage=500&forums=%s", page, forumList)
		jsonresp := basicAuth(path)
		var response Response
		if json.Valid(jsonresp) {
			// fmt.Println("Valid Json")
			err := json.Unmarshal(jsonresp, &response)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			// fmt.Println("Invalid Json")
			log.Fatal("Invalid Json")
		}
		// fmt.Printf("Processing Page %d of %d\n", response.Page, response.TotalPages)
		progress.SetValue((float64(response.Page)/float64(response.TotalPages))*100)
		for _, topic := range response.Results {
			if topic.FirstPost.Date.Before(end) && topic.LastPost.Date.After(begin) && topic.Prefix == "ic" {
				valid[topic.Id] = topic.Title
			}
		}
		// TODO: find good check for pages without valid posts.
		if page >= response.TotalPages {
			getNext = false
		}
	}
	progress.Hide()
	return valid
}

func GetValidPosts(begin time.Time, end time.Time, topics map[int]string, uid int, progress *widget.ProgressBar) []Post {
	type Response struct {
		Page int
		PerPage int
		TotalResults int
		TotalPages int
		Results []Post `json:"results"`
	}
	var user string = ""
	if uid != 0 {
		// fmt.Printf("For User:  %d\n", uid)
		user = fmt.Sprintf("&authors=%d", uid)
	}
	var page int = 0
	var getNext bool = true
	posts := []Post{}

	for getNext {
		page += 1
		// fmt.Printf("Processing Page %d\n", page)
		path := fmt.Sprintf("forums/posts/?sortBy=date&sortDir=desc&archived=0&hidden=0&page=%d&perPage=200&forums=%s%s", page, forumList, user)
		jsonresp := basicAuth(path)
		var response Response
		if json.Valid(jsonresp) {
			// fmt.Println("Valid Json")
			err := json.Unmarshal(jsonresp, &response)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			// fmt.Println("Invalid Json")
			log.Fatal("Invalid Json")
		}
		// fmt.Printf("of %d\n", response.TotalPages)
		progress.SetValue((float64(page)/15)*100)
		for _, post := range response.Results {
			if _, ok := topics[post.Item_id]; post.Date.Before(end) && post.Date.After(begin) && ok {
				posts = append(posts, post)
			}
			if post.Date.Before(begin) {
				getNext = false
				break
			}
		}
		if page >= response.TotalPages {
			getNext = false
			// fmt.Println("Max Page Exceeded")
		}
	}
	return posts
}

func CountMapToStruct(countMap map[int]map[int][]Post, topicMap map[int]string, userMap map[int]Member) []UserCount {
	var counts []UserCount
	for uid, user := range userMap {
		var topicCounts []TopicCount
		for tid, tposts := range countMap[uid] {
			tCount := TopicCount {
				Id: tid,
				Title: topicMap[tid],
				Count: len(tposts),
				Url: tposts[len(tposts)-1].Url,
			}
			topicCounts = append(topicCounts, tCount)
		}
		sort.Slice(topicCounts[:], func(i, j int) bool {
			return strings.ToLower(topicCounts[i].Title) < strings.ToLower(topicCounts[j].Title)
		})
		uCount := UserCount {
			Id: uid,
			Name: user.Name,
			Topics: topicCounts,
		}
		counts = append(counts, uCount)
	}
	sort.Slice(counts[:], func(i, j int) bool {
		return strings.ToLower(counts[i].Name) < strings.ToLower(counts[j].Name)
	})
	return counts
}
func GetCounts(prevMonth time.Time, userId int, progress *widget.ProgressBar, status *widget.Label) []UserCount {
	begin := BeginningOfMonth(prevMonth)
	end := EndOfMonth(prevMonth)
	// fmt.Println("Gathering Valid Topics")
	status.SetText("Gathering Valid Topics (This May Take Awhile)")
	status.Show()
	topics := GetValidTopics(begin, end, progress)
	// fmt.Printf("%d Topics Found\n", len(topics))
	// fmt.Println("Gathering Valid Posts")
	status.SetText("Gathering Valid Posts")
	progress.SetValue(0)
	progress.Show()
	posts := GetValidPosts(begin, end, topics, userId, progress)
	// fmt.Printf("%d total Posts in range\n", len(posts))
	status.SetText(fmt.Sprintf("%d total Posts in range\n", len(posts)))
	totalPosts := float64(len(posts))
	userThreads := make(map[int]map[int][]Post)
	users := make(map[int]Member)
	progress.SetValue(0)
	for i, post := range posts {
		postCount := (float64(i + 1)/totalPosts)*100
		// fmt.Printf("Processing: %d%% complete.\n", int(postCount))
		progress.SetValue(postCount)
		users[post.Author.Id] = post.Author
		if userThreads[post.Author.Id] == nil {
			userThreads[post.Author.Id] = map[int][]Post{}
		}
		userThreads[post.Author.Id][post.Item_id] = append(userThreads[post.Author.Id][post.Item_id], post)
	}
	postCounts := CountMapToStruct(userThreads, topics, users)
	// fmt.Println("Posts Counted.")
	status.SetText("Posts Counted.")
	// progress.Hide()
	return postCounts
}

func WriteToFile(counts []UserCount, prevMonth time.Time, status *widget.Label) {
	// fmt.Println("Writing to File")
	tmpl, err := template.New("html").Parse("<!DOCTYPE html><html><body>{{ range . }}<h4>{{ .Name }}</h4><ul>{{ range .Topics }}<li><a href='{{ .Url }}'>{{ .Title }}</a>:  {{.Count}} Posts</li>{{ end }}</ul>{{ end }}</body></html>")
	if err != nil {
		log.Fatal(err)
	}
	filename := fmt.Sprintf("%s-counts.html", prevMonth.Format("January-2006"))
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	tmpl.Execute(f, counts)
	// fmt.Printf("File: %s saved.", filename)
	status.SetText(fmt.Sprintf("File: %s saved.", filename))

}

func WriteToString(counts []UserCount) string {
	tmpl, err := template.New("html").Parse("{{ range . }}<h4>{{ .Name }}</h4><ul>{{ range .Topics }}<li><a href='{{ .Url }}'>{{ .Title }}</a>:  {{.Count}} Posts</li>{{ end }}</ul>{{ end }}")
	if err != nil {
		log.Fatal(err)
	}
	buf := &bytes.Buffer{}
	tmpl.Execute(buf, counts)
	return buf.String()
}

func SendAsMessage(msg string, uid int, prevMonth time.Time, status *widget.Label) {
	form := url.Values{}
	form.Add("from", strconv.Itoa(3446340))
	form.Add("to", strconv.Itoa(uid))
	form.Add("title", "Post counts for " + prevMonth.Format("January 2006"))
	form.Add("body", msg)

	var url string = "http://www.freedomplaybypost.com/api/core/messages"
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth(apiKey, "")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
	    status.SetText("Count Message Sent")
	}
}

func PostToRefcave(counts []UserCount, prevMonth time.Time, status *widget.Label) {
	body := WriteToString(counts)
	form := url.Values{}
	form.Add("forum", "19")
	form.Add("author", "3446340")
	form.Add("title", prevMonth.Format("January 2006") + " Post Counts")
	form.Add("post", body)
	form.Add("prefix", "Post Counts")

	var url string = "http://www.freedomplaybypost.com/api/forums/topics"
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth(apiKey, "")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
	    status.SetText("Counts Posted!")
	}
}

func main() {
	var userId int = 0
	layout := "2006-01-02T15:04:05.000Z"
	str := "2018-01-08T11:45:26.371Z"
	t, err := time.Parse(layout, str)

	if err != nil {
	    log.Fatal(err)
	}

	t = time.Now()
	prevMonth := t.UTC().AddDate(0,-1,0)

	var saveFile bool = false
	app := app.New()
	uidEntry := widget.NewEntry()
	uidEntry.SetPlaceHolder("User ID")
	status := widget.NewLabel("Status")
	progress := widget.NewProgressBar()
	progress.Max = 100
	progress.SetValue(0)
	progress.Hide()
	status.Hide()
	fileStatus := widget.NewLabel("Saving Counts to File")
	fileStatus.Hide()

	win := app.NewWindow("FCPBP Post Counter")
	win.CenterOnScreen()
	win.Resize(fyne.NewSize(400, 100))
	win.SetContent(widget.NewVBox(
			widget.NewHBox(
				widget.NewLabel("Currently: " + t.UTC().Format("January 2006")),
				widget.NewLabel("Counting for " + prevMonth.Format("January 2006")),
			),
			widget.NewHBox(
				widget.NewCheck("Save File", func(checked bool) {
					saveFile = checked
				}),
				uidEntry,
			),
			status,
			progress,
			fileStatus,
			widget.NewButton(fmt.Sprintf("Count Posts"), func() {
				status.Hide()
				fileStatus.Hide()
				status.SetText("Status")
				progress.SetValue(0)
				progress.Hide()
				validUID := false
				userId, err = strconv.Atoi(uidEntry.Text)
				if err != nil {
					if uidEntry.Text == "" {
						userId = 0
						validUID = true
					} else {
						status.SetText("Please Enter a Valid User ID")
						status.Show()
						uidEntry.SetText("")
					}
				} else {
					validUID = true
				}
				if validUID {
					postCounts := GetCounts(prevMonth, userId, progress, status)
					if saveFile {
						fileStatus.Show()
						WriteToFile(postCounts, prevMonth, fileStatus)
						uidEntry.SetText("")
					} else {
						fileStatus.SetText("Posting To Ref Cave")
						fileStatus.Show()
						PostToRefcave(postCounts, prevMonth, fileStatus)
					}
				}
			}),
			widget.NewButton("Quit", func() {
				app.Quit()
			}),
	))
	win.ShowAndRun()
}
