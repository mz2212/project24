package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/boltdb/bolt"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

type post struct {
	Name       string
	Subject    string
	Body       template.HTML
	TimePosted time.Time
	Comments   comments
	ThreadID   int
}

type comment struct {
	Name       string
	Subject    string
	Body       template.HTML
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
	p.DB = db // Change the line below to change how long to wait to poll
	tick := time.NewTicker(time.Minute * 30)
	go func() {
		for range tick.C {
			p.DB.Update(checkDel)
		}
	}()
	p.DB.Update(checkDel)
	//p.newPost(1, "Anon", "Test Post", "Test Post body that is a lot of words")
	srv := &http.Server{Addr: ":8080"}
	http.HandleFunc("/", p.viewPosts)
	http.HandleFunc("/newthread/", p.newThread)
	http.HandleFunc("/reply/", p.reply)
	http.HandleFunc("/view/", p.viewThread)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	go srv.ListenAndServe()
	fmt.Println("Server started. Press Ctrl-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc // This waits for somthing to come in on the "sc" channel.
	fmt.Println("[Main] Ctrl-C Recieved. Exiting!")
	tick.Stop()
	srv.Shutdown(nil)
	db.Close()
}

func i2b(i uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, i)
	return b
}

func checkDel(tx *bolt.Tx) error {
	b := tx.Bucket([]byte("posts"))
	pipe := new(bytes.Buffer)
	dec := gob.NewDecoder(pipe)
	var post post
	cur := b.Cursor()
	for k, v := cur.First(); k != nil; k, v = cur.Next() {
		pipe.Write(v)
		dec.Decode(&post) // Change the line below to change how long posts live
		if time.Since(post.TimePosted) >= 24*time.Hour {
			b.Delete(k)
		}
	}
	return nil
}

// Probably a bit much, but it makes it easier if I need to change it later
func parseAndSanitize(b []byte) template.HTML {
	return template.HTML(bluemonday.UGCPolicy().SanitizeBytes(blackfriday.Run(b)))
}
