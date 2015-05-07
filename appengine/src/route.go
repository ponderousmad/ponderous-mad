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
        http.HandleFunc("/post", post)
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

func post(w http.ResponseWriter, r *http.Request) {
        c := appengine.NewContext(r)
        post := Post{
                Content: r.FormValue("content"),
                Date:    time.Now(),
        }
        key := datastore.NewIncompleteKey(c, "Post", nil)
        _, err := datastore.Put(c, key, &post)
        if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
        }
        http.Redirect(w, r, "/", http.StatusFound)
}
