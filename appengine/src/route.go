package guestbook

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/urlfetch"
)

type Post struct {
	Content string `datastore:",noindex"`
	Date    time.Time
}

func init() {
	http.HandleFunc("/", root)
	http.HandleFunc("/projects.html", projects)
	http.HandleFunc("/.well-known/acme-challenge/", letsencrypt)
	http.HandleFunc("/captures/", captures)
}

func showPage(page *template.Template, path string, w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	q := datastore.NewQuery("Post").Order("-Date").Limit(10)
	posts := make([]Post, 0, 10)
	if _, err := q.GetAll(c, &posts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := page.ExecuteTemplate(w, path, posts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func root(w http.ResponseWriter, r *http.Request) {
	showPage(rootTemplate, "index.html", w, r)
}

func projects(w http.ResponseWriter, r *http.Request) {
	showPage(projectsTemplate, "projects.html", w, r)
}

func split_newline(s string) []string {
	return strings.Split(strings.Replace(s, "\r", "", -1), "\n")
}

var rootTemplate = template.Must(template.New("index").Funcs(template.FuncMap{"split": split_newline}).ParseFiles("html/index.html"))
var projectsTemplate = template.Must(template.New("projects").Funcs(template.FuncMap{"split": split_newline}).ParseFiles("html/projects.html"))


func letsencrypt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	challenge := ""
	response := ""
	if strings.HasSuffix(r.URL.Path, challenge) {
		w.Write([]byte(response))
	}
}

type CaptureEntry struct {
	id   string
	size int
}

var captureIDs map[string]CaptureEntry = make(map[string]CaptureEntry)
var captureNames = []string{}

func setupCaptureIDs(w http.ResponseWriter) {
	if len(captureIDs) != 0 {
		return
	}

	rand.Seed(time.Now().UTC().UnixNano())

	captureIDsPath := "./capture_ids/"
	files, _ := ioutil.ReadDir(captureIDsPath)
	for _, f := range files {
		filePath := path.Join(captureIDsPath, f.Name())
		contents, readErr := ioutil.ReadFile(filePath)
		if readErr != nil {
			http.Error(w, "Error reading file: "+readErr.Error(), http.StatusInternalServerError)
		} else {
			csvReader := csv.NewReader(strings.NewReader(string(contents)))
			data, parseErr := csvReader.ReadAll()
			if parseErr != nil {
				http.Error(w, "Error parsing file: "+parseErr.Error(), http.StatusInternalServerError)
			} else {
				for _, entry := range data {
					imageName := url.QueryEscape(strings.ToLower(entry[0]))
					size, _ := strconv.Atoi(entry[2])
					captureIDs[imageName] = CaptureEntry{
						id:   entry[1],
						size: size,
					}
					captureNames = append(captureNames, imageName)
				}
			}
		}
	}
}

func captures(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)

	setupCaptureIDs(w)

	pathParts := strings.Split(r.URL.Path, "/")
	imageName := pathParts[2]

	if imageName == "random.png" {
		imageName = captureNames[rand.Intn(len(captureNames))]
	} else {
		imageName = strings.ToLower(imageName)
		if strings.ContainsAny(imageName, " ") {
			imageName = url.QueryEscape(imageName)
		}
	}

	captureEntry, lookupOk := captureIDs[imageName]
	if lookupOk {
		baseURL := "https://drive.google.com/uc?export=download&id="
		response, fetchErr := client.Get(baseURL + captureEntry.id)
		if fetchErr != nil {
			http.Error(w, fetchErr.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", response.Header["Content-Type"][0])

		defer response.Body.Close()
		_, writeErr := io.Copy(w, response.Body)
		if writeErr != nil {
			http.Error(w, writeErr.Error(), http.StatusInternalServerError)
		}
	} else {
		fmt.Fprintf(w, "Requested image not found: %s\n", imageName)
	}
}
