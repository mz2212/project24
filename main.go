package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"encoding/binary"

	"github.com/boltdb/bolt"
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

type posts struct {
	DB *bolt.DB
}

type request struct {
	R  *http.Request
	ID int
}

type comments map[int]comment

func main() {
	var p posts
	// Open the database
	db, err := bolt.Open("posts.db", 0644, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		fmt.Println("Failed to open DB: ", err)
	}
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("posts"))
		if err != nil {
			return err
		}
		return nil
	})
	p.DB = db
	//p.newPost(1, "Anon", "Test Post", "Test Post body that is a lot of words")
	http.HandleFunc("/", p.viewPosts)
	http.HandleFunc("/newthread/", p.newThread)
	http.HandleFunc("/reply/", p.reply)
	http.HandleFunc("/view/", p.viewThread)
	http.ListenAndServe(":8080", nil)
}

// I know deleting posts before viewing probably isn't the best way to do it, but it's better than polling
// Actually, polling every half hour or so might be better...
func (p *posts) viewPosts(w http.ResponseWriter, r *http.Request) {
	pipe := new(bytes.Buffer)
	dec := gob.NewDecoder(pipe)
	posts := []post{}
	var post post
	t, err := template.ParseFiles("index.html")
	if err != nil {
		fmt.Println("Failed to load template: ", err)
	}
	p.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("posts"))
		cur := b.Cursor()
		for k, v := cur.First(); k != nil; k, v = cur.Next() {
			pipe.Write(v)
			dec.Decode(&post)
			posts = append(posts, post)
		}
		return nil
	})
	t.Execute(w, posts)
}

func (p *posts) viewThread(w http.ResponseWriter, r *http.Request) {
	pipe := new(bytes.Buffer)
	dec := gob.NewDecoder(pipe)
	var post post
	threadID, err := strconv.Atoi(r.URL.Path[len("/view/"):])
	if err != nil {
		fmt.Println("Failed to show thread: ", err)
	}
	t, err := template.ParseFiles("view.html")
	if err != nil {
		fmt.Println("Failed to load template: ", err)
	}
	p.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("posts"))
		pipe.Write(b.Get(i2b(uint64(threadID))))
		dec.Decode(&post)
		return nil
	})
	t.Execute(w, post)
}

func (p *posts) newThread(w http.ResponseWriter, r *http.Request) {
	var req request
	req.R = r
	p.DB.Update(req.newPost)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (p *posts) reply(w http.ResponseWriter, r *http.Request) {
	var req request
	req.R = r
	threadID, err := strconv.Atoi(r.URL.Path[len("/reply/"):])
	if err != nil {
		fmt.Println("Failed attempt to post comment: ", err)
	}
	req.ID = threadID
	err = p.DB.Update(req.newComment)
	http.Redirect(w, r, "/view/"+r.URL.Path[len("/reply/"):], http.StatusFound)
}

func (r request) newPost(tx *bolt.Tx) error {
	pipe := new(bytes.Buffer)
	enc := gob.NewEncoder(pipe)
	b, err := tx.CreateBucketIfNotExists([]byte("posts"))
	if err != nil {
		return err
	}
	id, _ := b.NextSequence()
	enc.Encode(post{
		Name:       r.R.FormValue("name"),
		Subject:    r.R.FormValue("subject"),
		Body:       r.R.FormValue("body"),
		TimePosted: time.Now(),
		Comments:   make(comments),
		ThreadID:   int(id),
	})
	err = b.Put(i2b(id), pipe.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (r request) newComment(tx *bolt.Tx) error {
	var post post
	pipe := new(bytes.Buffer)
	enc := gob.NewEncoder(pipe)
	dec := gob.NewDecoder(pipe)
	b := tx.Bucket([]byte("posts"))
	pipe.Write(b.Get(i2b(uint64(r.ID))))
	dec.Decode(&post)
	comment := comment{
		Name:       r.R.FormValue("name"),
		Subject:    post.Subject,
		Body:       r.R.FormValue("body"),
		TimePosted: time.Now(),
	}
	post.Comments[len(post.Comments)+1] = comment
	enc.Encode(post)
	err := b.Put(i2b(uint64(r.ID)), pipe.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func i2b(i uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, i)
	return b
}
