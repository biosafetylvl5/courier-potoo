package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTagOf(t *testing.T) {
	cases := map[string]string{
		"aB3dE6gH.png": "aB3dE6gH",
		"aB3dE6gH.gif": "aB3dE6gH",
		"noext":        "noext",
	}
	for in, want := range cases {
		if got := tagOf(in); got != want {
			t.Errorf("tagOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCleanTag(t *testing.T) {
	for _, tag := range []string{"aB3dE6gH", "with-dash_and.dot"} {
		if _, ok := cleanTag(tag); !ok {
			t.Errorf("cleanTag(%q) rejected a valid tag", tag)
		}
	}
	for _, tag := range []string{"", ".", "..", "../etc/passwd", `..\windows`, "a/b", `a\b`, "C:foo"} {
		if _, ok := cleanTag(tag); ok {
			t.Errorf("cleanTag(%q) accepted an unsafe tag", tag)
		}
	}
}

// newTestServer creates an output dir holding done.png and pending.png, and a
// gif dir holding only done.gif.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	s := &server{outputDir: t.TempDir(), gifDir: t.TempDir()}
	write := func(dir, name string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(s.outputDir, "done.png")
	write(s.outputDir, "pending.png")
	write(s.gifDir, "done.gif")

	ts := httptest.NewServer(s.routes())
	t.Cleanup(ts.Close)
	return ts
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, string(body)
}

func TestIndex(t *testing.T) {
	ts := newTestServer(t)
	code, body := get(t, ts.URL+"/")
	if code != 200 || !strings.HasPrefix(body, "<!DOCTYPE html>") {
		t.Errorf("GET / = %d, body starts %q", code, body[:min(len(body), 20)])
	}
}

func TestImages(t *testing.T) {
	ts := newTestServer(t)
	_, body := get(t, ts.URL+"/images")
	for _, tag := range []string{"done", "pending"} {
		if !strings.Contains(body, "<p>"+tag+"</p>") {
			t.Errorf("/images missing tag %q", tag)
		}
	}
	if strings.Count(body, "background-color: green") != 1 {
		t.Errorf("/images should mark exactly one tile processed:\n%s", body)
	}
}

func TestImage(t *testing.T) {
	ts := newTestServer(t)
	cases := map[string]int{
		"done":          200,
		"nope":          404,
		"":              400,
		"../etc/passwd": 400,
	}
	for tag, want := range cases {
		if code, _ := get(t, ts.URL+"/image?tag="+tag); code != want {
			t.Errorf("/image?tag=%q = %d, want %d", tag, code, want)
		}
	}
}

func TestGif(t *testing.T) {
	ts := newTestServer(t)
	if _, body := get(t, ts.URL+"/gif?tag=done"); !strings.Contains(body, "/gifs/done.gif") {
		t.Errorf("/gif?tag=done should link the GIF:\n%s", body)
	}
	for _, tag := range []string{"pending", "../etc/passwd"} {
		if _, body := get(t, ts.URL+"/gif?tag="+tag); !strings.Contains(body, "Please select") {
			t.Errorf("/gif?tag=%q should show the placeholder:\n%s", tag, body)
		}
	}
	if code, _ := get(t, ts.URL+"/gifs/done.gif"); code != 200 {
		t.Errorf("/gifs/done.gif = %d, want 200", code)
	}
}
