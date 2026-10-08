package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
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

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

//go:embed pages/index.html
var indexHTML []byte

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

// server holds the input images written by potoo.py (outputDir) and the
// GIFs Courier generates from them (gifDir).
type server struct {
	outputDir string
	gifDir    string
}

// tagOf strips the extension from a filename, e.g. "aB3dE6gH.png" -> "aB3dE6gH".
func tagOf(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}

// cleanTag rejects tags that could escape the data directories.
func cleanTag(tag string) (string, bool) {
	if tag == "" || tag == "." || tag == ".." || strings.ContainsAny(tag, `/\:`) {
		return "", false
	}
	return tag, true
}

func (s *server) gifPath(tag string) string {
	return filepath.Join(s.gifDir, tag+".gif")
}

func (s *server) isProcessed(tag string) bool {
	_, err := os.Stat(s.gifPath(tag))
	return err == nil
}

// findInput returns the path of the input image for a tag, whatever its extension.
func (s *server) findInput(tag string) (string, error) {
	entries, err := os.ReadDir(s.outputDir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() && tagOf(entry.Name()) == tag {
			return filepath.Join(s.outputDir, entry.Name()), nil
		}
	}
	return "", fs.ErrNotExist
}

func (s *server) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write(indexHTML); err != nil {
			log.Println(err)
		}
	})

	mux.HandleFunc("/images", func(w http.ResponseWriter, r *http.Request) {
		entries, err := os.ReadDir(s.outputDir)
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
				Processed: s.isProcessed(tag),
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

	mux.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		tag, ok := cleanTag(r.URL.Query().Get("tag"))
		if !ok {
			http.Error(w, "Missing or invalid tag", 400)
			return
		}

		path, err := s.findInput(tag)
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

	mux.Handle("/gifs/",
		http.StripPrefix("/gifs/", http.FileServer(http.Dir(s.gifDir))),
	)

	mux.HandleFunc("/gif", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tag, ok := cleanTag(r.URL.Query().Get("tag"))
		if !ok || !s.isProcessed(tag) {
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

	return mux
}

// envOr returns the environment variable key, or fallback if it is unset.
func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func main() {
	outputDir := flag.String("output-dir", envOr("POTOO_OUTPUT_DIR", "/output_dir"),
		"directory of input images (env POTOO_OUTPUT_DIR)")
	gifDir := flag.String("gif-dir", envOr("POTOO_GIF_DIR", "/gif_dir"),
		"directory of generated GIFs (env POTOO_GIF_DIR)")
	addr := flag.String("addr", envOr("POTOO_ADDR", ":8080"),
		"address to listen on (env POTOO_ADDR)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("potoo-server", version)
		return
	}

	s := &server{outputDir: *outputDir, gifDir: *gifDir}
	log.Printf("potoo-server %s: serving %s and %s on %s", version, s.outputDir, s.gifDir, *addr)
	log.Fatal(http.ListenAndServe(*addr, s.routes()))
}
