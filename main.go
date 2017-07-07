package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
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

type comment struct {
	Name       string
	Subject    string
	Body       string
	TimePosted time.Time
}

type posts map[int]post
type comments map[int]comment

func main() {
	var (
		p = make(posts)
	) // Change below to change poll time
	tick := time.NewTicker(time.Minute * 30)
	go func() {
		for range tick.C {
			for key, po := range p { // The line below is what needs to be changed to change the time posts live
				if time.Since(po.TimePosted) >= 24*time.Hour {
					delete(p, key)
				}
			}
		}
	}()
	//p.newPost(1, "Anon", "Test Post", "Test Post body that is a lot of words")
	srv := &http.Server{Addr: ":8080"}
	http.HandleFunc("/", p.viewPosts)
	http.HandleFunc("/newthread/", p.newThread)
	http.HandleFunc("/reply/", p.reply)
	http.HandleFunc("/view/", p.viewThread)
	go srv.ListenAndServe()
	fmt.Println("Server started. Press Ctrl-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc // This waits for somthing to come in on the "sc" channel.
	fmt.Println("Ctrl-C Recieved. Exiting!")
	tick.Stop()
	srv.Shutdown(nil)
}

func (p *posts) viewPosts(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("index.html")
	if err != nil {
		fmt.Println("Failed to load template: ", err)
	}
	t.Execute(w, p)
}

func (p *posts) viewThread(w http.ResponseWriter, r *http.Request) {
	threadID, err := strconv.Atoi(r.URL.Path[len("/view/"):])
	if err != nil {
		fmt.Println("Failed to show thread: ", err)
	}
	t, err := template.ParseFiles("view.html")
	if err != nil {
		fmt.Println("Failed to load template: ", err)
	}
	t.Execute(w, (*p)[threadID])
}

func (p *posts) newThread(w http.ResponseWriter, r *http.Request) {
	p.newPost((*p)[len(*p)].ThreadID+1, r.FormValue("name"), r.FormValue("subject"), r.FormValue("body"))
	http.Redirect(w, r, "/", http.StatusFound)
}

func (p *posts) reply(w http.ResponseWriter, r *http.Request) {
	threadID, err := strconv.Atoi(r.URL.Path[len("/reply/"):])
	if err != nil {
		fmt.Println("Failed attempt to post comment: ", err)
	}
	p.newComment(threadID, r.FormValue("name"), r.FormValue("body"))
	http.Redirect(w, r, "/view/"+r.URL.Path[len("/reply/"):], http.StatusFound)
}

func (p *posts) newPost(threadID int, name, subject, body string) {
	(*p)[threadID] = post{ThreadID: threadID, Name: name, Subject: subject, Body: body, TimePosted: time.Now(), Comments: make(comments)}
}

func (p *posts) newComment(threadID int, name, body string) {
	(*p)[threadID].Comments[len((*p)[threadID].Comments)+1] = comment{Name: name, Subject: (*p)[threadID].Subject, Body: body, TimePosted: time.Now()}
}
