package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/boltdb/bolt"
)

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
