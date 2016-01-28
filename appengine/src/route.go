package guestbook

import (
        "html/template"
        "net/http"
        "time"
        "strings"

        "appengine"
        "appengine/datastore"
)

type Post struct {
        Content string `datastore:",noindex"`
        Date    time.Time
}

func init() {
        http.HandleFunc("/", root)
        http.HandleFunc("/.well-known/acme-challenge/", letsencrypt)
}

func root(w http.ResponseWriter, r *http.Request) {
        c := appengine.NewContext(r)
        q := datastore.NewQuery("Post").Order("-Date").Limit(10)
        posts := make([]Post, 0, 10)
        if _, err := q.GetAll(c, &posts); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
        }
        if err := postsTemplate.ExecuteTemplate(w, "index.html", posts); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
        }
}

func split_newline(s string) []string {
    return strings.Split(strings.Replace(s, "\r", "", -1), "\n")
}

var postsTemplate = template.Must(template.New("index").Funcs(template.FuncMap{"split": split_newline,}).ParseFiles("html/index.html"))

func letsencrypt(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/plain")
        challenge := ""
        response := ""
        if strings.HasSuffix(r.URL.Path, challenge) {
            w.Write([]byte(response))
        }
}
