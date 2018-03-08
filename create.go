package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
)

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
		Body:       parseAndSanitize([]byte(r.R.FormValue("body"))),
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
		Body:       parseAndSanitize([]byte(r.R.FormValue("body"))),
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
