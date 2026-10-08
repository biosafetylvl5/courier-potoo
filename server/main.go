package main

import (
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Input images written by potoo.py, and the GIFs Courier generates from them.
const (
	outputDir = "/output_dir"
	gifDir    = "/gif_dir"
)

var galleryTemplate = template.Must(template.New("gallery").Parse(`
{{range .}}
<div style="background-color: {{if .Processed}}green{{else}}transparent{{end}}"
	hx-get="/gif?tag={{urlquery .Tag}}"
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
	modTime   time.Time
}

// tagOf strips the extension from a filename, e.g. "aB3dE6gH.png" -> "aB3dE6gH".
func tagOf(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}

// cleanTag rejects tags that could escape the data directories.
func cleanTag(tag string) (string, bool) {
	if tag == "" || tag != filepath.Base(tag) || tag == "." || tag == ".." {
		return "", false
	}
	return tag, true
}

func gifPath(tag string) string {
	return filepath.Join(gifDir, tag+".gif")
}

func isProcessed(tag string) bool {
	_, err := os.Stat(gifPath(tag))
	return err == nil
}

// findInput returns the path of the input image for a tag, whatever its extension.
func findInput(tag string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(outputDir, tag+".*"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fs.ErrNotExist
	}
	return matches[0], nil
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
		entries, err := os.ReadDir(outputDir)
		if err != nil {
			http.Error(w, "Error reading images", 500)
			return
		}

		var images []Image
		for _, entry := range entries {
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				// File was removed between ReadDir and Info.
				continue
			}
			tag := tagOf(entry.Name())
			images = append(images, Image{
				Tag:       tag,
				Processed: isProcessed(tag),
				modTime:   info.ModTime(),
			})
		}

		// Newest first, so tiles stay put between polls.
		sort.Slice(images, func(i, j int) bool {
			if images[i].modTime.Equal(images[j].modTime) {
				return images[i].Tag < images[j].Tag
			}
			return images[i].modTime.After(images[j].modTime)
		})

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := galleryTemplate.Execute(w, images); err != nil {
			log.Println(err)
		}
	})

	http.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		tag, ok := cleanTag(r.URL.Query().Get("tag"))
		if !ok {
			http.Error(w, "Missing or invalid tag", 400)
			return
		}

		path, err := findInput(tag)
		if errors.Is(err, fs.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "Error reading image", 500)
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, path)
	})

	http.Handle("/gifs/",
		http.StripPrefix("/gifs/", http.FileServer(http.Dir(gifDir))),
	)

	http.HandleFunc("/gif", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tag, ok := cleanTag(r.URL.Query().Get("tag"))
		if !ok || !isProcessed(tag) {
			if err := displayedGif.Execute(w, ""); err != nil {
				log.Println(err)
			}
			return
		}

		if err := displayedGif.Execute(w, "/gifs/"+tag+".gif"); err != nil {
			http.Error(w, "Error formatting gif", 500)
			return
		}
	})

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
