// Package main contains a simple file uploader
package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	downloadValue = "{DOWNLOAD}"
	isUpload      = "/store"
	indexHTML     = `<!doctype html>
<html lang="en">
<head>
<meta charset="UTF-8">
<style>
.entry
{
    margin-left: auto;
    margin-right: auto ;
    border: 1px solid black;
    border-radius: 10px;
    border-style: none none solid solid;
    padding: 5px;
}

body
{
    font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
    font-size: 20px;
}

#main
{
    width: 80%;
    margin-left: auto;
    margin-right: auto;
    padding: 20px;
}

.small
{
    font-size: 10px;
}

#site
{
    margin-left: auto;
    margin-right: auto;
    width: 70%;
}
</style>
<title>ttypty</title>
</head>
<body>
  <div id="site">
    <div id="main">
    <form
      id="form"
      enctype="multipart/form-data"
      action="/storeto"
      method="POST"
    >
      <input class="input file-input" type="file" name="file" multiple />
      <button class="button" type="submit">Submit</button>
    </form>
    <hr />
    <br />
{{range $idx, $file := .Files }}
    {{ $file }}
    <br />
{{end }}
    <br />
    <br />
    <hr />
<a href="/` + downloadValue + `"><div class="entry">files</div></a>
    </div>
  </div>
</body>
</html>`
)

// Config handles tool configuration
type Config struct {
	Bind       string
	Store      string
	Extensions []string
}

func uploadHandler(store string, extensions []string, r *http.Request) error {
	if r.Method != "POST" {
		return nil
	}

	// 32 MB is the default used by FormFile()
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return err
	}

	files := r.MultipartForm.File["file"]
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			return err
		}

		defer file.Close()
		name := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(string(fileHeader.Filename), " ", "-")))
		for _, ext := range extensions {
			if !strings.HasSuffix(name, fmt.Sprintf(".%s", ext)) {
				continue
			}
			hasher := sha256.New()
			if _, err := hasher.Write([]byte(name)); err != nil {
				return err
			}

			dt := time.Now().Format("02.T_150405")
			h := hex.EncodeToString(hasher.Sum(nil))[0:7]
			name = fmt.Sprintf("%s.%s.%s", dt, h, ext)
			break
		}
		target := filepath.Join(store, name)
		f, err := os.Create(target)
		if err != nil {
			return err
		}

		defer f.Close()

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func onError(text string, err error) {
	fmt.Fprintf(os.Stderr, "%s (%v)", text, err)
}

func run() error {
	home := os.Getenv("HOME")
	read, err := os.ReadFile(filepath.Join(home, ".config", "etc", "uploads"))
	if err != nil {
		return err
	}
	var cfg Config
	if err := json.Unmarshal(read, &cfg); err != nil {
		return err
	}
	store := filepath.Join(home, cfg.Store)
	downloadName := strings.ToLower(cfg.Store)
	t, err := template.New("t").Parse(strings.Replace(indexHTML, downloadValue, downloadName, 1))
	if err != nil {
		return err
	}
	router := http.NewServeMux()
	router.HandleFunc(isUpload+"to", func(w http.ResponseWriter, r *http.Request) {
		if err := uploadHandler(store, cfg.Extensions, r); err != nil {
			onError("upload error", err)
		}
		http.Redirect(w, r, isUpload, http.StatusSeeOther)
	})
	router.HandleFunc(isUpload, func(w http.ResponseWriter, _ *http.Request) {
		if err := t.Execute(w, nil); err != nil {
			onError("template error", err)
		}
	})
	prefix := fmt.Sprintf("/%s/", downloadName)
	router.Handle(prefix, http.StripPrefix(prefix, http.FileServer(http.Dir(store))))
	s := &http.Server{
		Addr:    cfg.Bind,
		Handler: router,
	}
	return s.ListenAndServe()
}
