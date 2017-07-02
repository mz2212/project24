package main

import (
	"fmt"
	"html/template"
	"net/http"
	"time"
)

type post struct {
	ThreadID   int
	Name       string
	Subject    string
	Body       string
	TimePosted time.Time
}

type posts []post

func main() {
	var (
		p posts
	)
	p.newPost(1, "Anon", "Test Post", "Test Post body that is a lot of words")
	http.HandleFunc("/", p.viewPosts)
	http.HandleFunc("/newthread/", p.newThread)
	http.ListenAndServe(":8080", nil)
}

func (p *posts) newPost(threadID int, name, subject, body string) {
	*p = append(*p, post{ThreadID: threadID, Name: name, Subject: subject, Body: body, TimePosted: time.Now()})
}

// I know deleting posts before viewing probably isn't the best way to do it, but it's better than polling
func (p *posts) viewPosts(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("index.html")
	if err != nil {
		fmt.Println("Failed to load template: ", err)
	}
	for i, po := range *p { // The line below is what needs to be changed to change the time posts live
		if time.Since(po.TimePosted) >= 24*time.Hour {
			*p = (*p)[:i+copy((*p)[i:], (*p)[i+1:])] // This is weird, but it works
		}
	}
	t.Execute(w, p)
}

func (p *posts) newThread(w http.ResponseWriter, r *http.Request) {
	p.newPost(len(*p)+1, r.FormValue("name"), r.FormValue("subject"), r.FormValue("body"))
	http.Redirect(w, r, "/", http.StatusFound)
}
