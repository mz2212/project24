package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"
)

type post struct {
	Name       string
	Subject    string
	Body       string
	TimePosted time.Time
	Comments   comments
	ThreadID   int
}

// I'm aware I could just use the `posts` type everywhere,
// but this is easier in my head.
type posts map[int]post
type comments map[int]post

func main() {
	var (
		p = make(posts)
	)
	p.newPost(1, "Anon", "Test Post", "Test Post body that is a lot of words")
	http.HandleFunc("/", p.viewPosts)
	http.HandleFunc("/newthread/", p.newThread)
	http.HandleFunc("/reply/", p.reply)
	http.ListenAndServe(":8080", nil)
}

// I know deleting posts before viewing probably isn't the best way to do it, but it's better than polling
// Actually, polling every half hour or so might be better...
func (p *posts) viewPosts(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("index.html")
	if err != nil {
		fmt.Println("Failed to load template: ", err)
	}
	for key, po := range *p { // The line below is what needs to be changed to change the time posts live
		if time.Since(po.TimePosted) >= 24*time.Hour {
			delete(*p, key)
		}
	}
	t.Execute(w, p)
}

func (p *posts) newThread(w http.ResponseWriter, r *http.Request) {
	p.newPost(len(*p)+1, r.FormValue("name"), r.FormValue("subject"), r.FormValue("body"))
	http.Redirect(w, r, "/", http.StatusFound)
}

func (p *posts) reply(w http.ResponseWriter, r *http.Request) {
	threadID, err := strconv.Atoi(r.URL.Path[len("/reply/"):])
	if err != nil {
		fmt.Println("Failed attempt to post comment: ", err)
	}
	p.newComment(threadID, r.FormValue("name"), r.FormValue("body"))
	http.Redirect(w, r, "/", http.StatusFound)
}

func (p *posts) newPost(threadID int, name, subject, body string) {
	(*p)[threadID] = post{ThreadID: threadID, Name: name, Subject: subject, Body: body, TimePosted: time.Now(), Comments: make(comments)}
}

func (p *posts) newComment(threadID int, name, body string) {
	(*p)[threadID].Comments[len((*p)[threadID].Comments)+1] = post{ThreadID: threadID, Name: name, Subject: (*p)[threadID].Subject, Body: body, TimePosted: time.Now(), Comments: make(comments)}
}

func (c *comments) newComment(threadID int, name, body string) {
	(*c)[threadID].Comments[len((*c)[threadID].Comments)+1] = post{ThreadID: threadID, Name: name, Subject: (*c)[threadID].Subject, Body: body, TimePosted: time.Now(), Comments: make(comments)}
}
