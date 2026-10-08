package main

import (
	"encoding/base64"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/redis/go-redis/v9"
)

var rdb = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

var galleryTemplate = template.Must(template.New("gallery").Parse(`
{{range .}}
<div style="background-color: {{if .Processed}}green{{else}}transparent{{end}}"
	hx-get="/gif?tag={{urlquery .Tag}}&processed={{urlquery .Processed}}"
	hx-trigger="click"
	hx-target="#displayed-gif"
	hx-swap="innerHTML">

	<img src="/image?tag={{urlquery .Tag}}" loading="lazy">
	<p>{{.Tag}}</p>
</div>
{{else}}
<p>No images found.</p>
{{end}}
`))

var displayedGif = template.Must(template.New("gif").Parse(`
{{if .}}
<img src={{.}} alt="Animation">
{{else}}
<p>Please select a processed image</p>
{{end}}
`))

type Image struct {
	Tag       string
	Processed bool
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "pages/index.html")
	})
	http.HandleFunc("/images", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var cursor uint64

		var images []Image

		for {
			keys, next, err := rdb.Scan(ctx, cursor, "*", 100).Result()
			if err != nil {
				http.Error(w, "Redis error", 500)
				return
			}

			for _, key := range keys {
				value, err := rdb.HGet(ctx, key, "processed").Result()
				if err == redis.Nil {
					value = "false"
				} else if err != nil {
					http.Error(w, "Redis error", 500)
					return
				}

				processed, err := strconv.ParseBool(value)
				if err != nil {
					http.Error(w, "Invalid boolean", 500)
					return
				}

				images = append(images, Image{
					Tag:       key,
					Processed: processed,
				})
			}

			cursor = next
			if cursor == 0 {
				break
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := galleryTemplate.Execute(w, images); err != nil {
			log.Println(err)
		}
	})

	http.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		tag := r.URL.Query().Get("tag")
		if tag == "" {
			http.Error(w, "Missing tag", 400)
			return
		}

		data, err := rdb.HGet(r.Context(), tag, "image").Result()
		if errors.Is(err, redis.Nil) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "Redis error", 500)
			return
		}

		img, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			http.Error(w, "Invalid base64", 500)
			return
		}

		w.Header().Set("Content-Type", http.DetectContentType(img))
		w.Header().Set("Cache-Control", "no-cache")

		if _, err := w.Write(img); err != nil {
			log.Println(err)
		}
	})

	http.Handle("/gifs/",
		http.StripPrefix("/gifs/", http.FileServer(http.Dir("../gif_dir"))),
	)

	http.HandleFunc("/gif", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		processed, err := strconv.ParseBool(query.Get("processed"))
		if err != nil {
			http.Error(w, "Bad query to /gif", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if !processed {
			if err := displayedGif.Execute(w, ""); err != nil {
				log.Println(err)
			}
			return
		}
		tag := r.URL.Query().Get("tag")

		if err := displayedGif.Execute(w, "/gifs/"+tag+".gif"); err != nil {
			http.Error(w, "Error formatting gif", 500)
			return
		}
	})

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
