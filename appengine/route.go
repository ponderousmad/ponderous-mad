package ponderousmad

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
	"appengine/urlfetch"
)

type handler func(w http.ResponseWriter, r *http.Request)

type Project struct {
	Name string
	Page string
}

func init() {
	http.HandleFunc("/", pageView("index"))
	http.HandleFunc("/vrkspace.html", pageView("vrkspace"))
	http.HandleFunc("/.well-known/acme-challenge/", letsencrypt)
	http.HandleFunc("/captures/", captures)

	projects := []Project{
		Project{"arcake", "index"},
		Project{"blow-up", "game"},
		Project{"blitblort-demo", "index"},
		Project{"c3d", "index"},
		Project{"combust-a-move", "game"},
		Project{"greyfield", "index"},
		Project{"lost-on-mars", "play"},
		Project{"markovio", "fourier"},
		Project{"opdozitz", "game"},
		Project{"pipevo", "game"},
		Project{"scrace", "game"},
		Project{"tapwords", "tapwords"},
		Project{"tojam11", "index"},
		Project{"tojam12", "index"},
		Project{"wavebreaker", "index"},
		Project{"wordevo", "evo"},
	}

	for _, project := range projects {
		http.HandleFunc("/" + project.Name + "/", projectPage(project))
		http.HandleFunc("/" + project.Name, projectPage(project))
	}
}

func pageView(name string) handler {
	tmpl, parseErr := template.ParseFiles(path.Join("html", name + ".html"))
	return func(w http.ResponseWriter, r *http.Request) {
		if parseErr != nil {
			http.Error(w, parseErr.Error(), http.StatusInternalServerError)
			return
		}
		if err := tmpl.ExecuteTemplate(w, name + ".html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func letsencrypt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	challenge := ""
	response := ""
	if strings.HasSuffix(r.URL.Path, challenge) {
		w.Write([]byte(response))
	}
}

func projectPage(project Project) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(r.URL.Path, "/")
		ctx := appengine.NewContext(r)
		ctx.Infof("Requested URL: %v", r.URL.Path)
		ctx.Infof("Parts: %v", pathParts)

		if (len(pathParts) < 3) {
			http.Redirect(w, r, "/" + project.Name + "/" + project.Page + ".html", 302)
			return
		}
		tmpl, parseErr := template.ParseFiles(path.Join("html","404.html"))
		if parseErr != nil {
			http.Error(w, parseErr.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		if err := tmpl.ExecuteTemplate(w, "404.html", project); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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
