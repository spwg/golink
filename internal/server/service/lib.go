// Package service provides an http service for go links.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

const (
	indexPage = `<html>
<body>
<p><b>Create a new link</b></p>
<form action="/new_golink" method="post">
<label for="name">Link name:</label><br>
<input required type="text" id="name" name="name"><br>
<label for="link">Link</label><br>
<input required type="url" id="link" name="link"><br>
<input type="submit">
</form>
<p><b>Manage links</b></p>
{{range .Links}}
<a href="/golink/{{.Name}}">{{.Name}}</a><br>
{{end}}
</body>
</html>`
	goLinkPage = `<html>
<body>
<p>Name: {{.Name}}</p>
<p>URL: <a href="/go/{{.Name}}">{{.Link}}</a></p>
</body>
</html>`
)

var (
	indexPageTemplate  = template.Must(template.New("").Parse(indexPage))
	goLinkPageTemplate = template.Must(template.New("").Parse(goLinkPage))
)

// GoLink is a service for shortened links.
type GoLink struct {
	db *sql.DB
}

type golink struct {
	Name string
	Link string
}

// New creates a *GoLink.
func New(db *sql.DB) *GoLink {
	return &GoLink{db}
}

// Run installs and starts up the service.
func (gl *GoLink) Run(ctx context.Context) error {
	gl.install(ctx)
	if err := gl.startUp(ctx); err != nil {
		return err
	}
	return nil
}

func (gl *GoLink) install(ctx context.Context) {
	http.HandleFunc("/", gl.indexHandler)
	http.HandleFunc("/create_golink", gl.createHandler)
	http.HandleFunc("/golink/", gl.readHandler)
	http.HandleFunc("/update_golink", gl.updateHandler)
	http.HandleFunc("/delete_golink", gl.deleteHandler)
	http.HandleFunc("/go/", gl.goHandler)
}

func (gl *GoLink) startUp(ctx context.Context) error {
	addr := fmt.Sprintf("localhost:%v", 10123)
	log.Printf("Server listening on http://%v", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		return fmt.Errorf("listen and serve failed: %v", err)
	}
	return nil
}

func (gl *GoLink) indexHandler(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	const query = "select name, url from links;"
	rows, err := gl.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Failed to query all links in the database: %v", err)
		http.Error(resp, "Failed to query all links in the database.", http.StatusInternalServerError)
		return
	}
	var links []*golink
	for rows.Next() {
		var name, url string
		if err := rows.Scan(&name, &url); err != nil {
			log.Printf("Failed to scan link: %v", err)
			http.Error(resp, "Failed to query all links in the databse.", http.StatusInternalServerError)
			return
		}
		links = append(links, &golink{name, url})
	}
	if err := indexPageTemplate.Execute(resp, struct {
		Links []*golink
	}{links}); err != nil {
		http.Error(resp, "Internal server error.", http.StatusInternalServerError)
		return
	}
}

func (gl *GoLink) createHandler(resp http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		log.Printf("Failed to parse form: %v", err)
		http.Error(resp, "Failed to parse form.", http.StatusBadRequest)
		return
	}
	ctx := req.Context()
	name := req.PostForm.Get("name")
	if name == "" {
		http.Error(resp, "The golink's name is missing, did you forget to write one?", http.StatusBadRequest)
		return
	}
	link := req.PostForm.Get("link")
	if link == "" {
		http.Error(resp, "The golink's link is missing, did you forget to include a URL?", http.StatusBadRequest)
		return
	}
	_, ok, err := gl.linkByName(ctx, name)
	if err != nil {
		log.Printf("Failed to query whether name=%q exists: %v", name, err)
		http.Error(resp, "Failed to save the link in the database.", http.StatusInternalServerError)
		return
	}
	if ok {
		msg := fmt.Sprintf("The golink %q already exists.", name)
		http.Error(resp, msg, http.StatusConflict)
		return
	}
	query := "insert into links (name, url) values (?, ?);"
	if _, err := gl.db.ExecContext(ctx, query, name, link); err != nil {
		log.Printf("Failed to insert name=%q link=%q in the database: %v", name, link, err)
		http.Error(resp, "Failed to save the link in the database.", http.StatusInternalServerError)
		return
	}
	log.Printf("Saved new link: %v -> %v", name, link)
	http.Redirect(resp, req, "/golink/"+name, http.StatusSeeOther)
}

func (gl *GoLink) readHandler(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	p := req.URL.EscapedPath()
	p = strings.TrimPrefix(p, "/")
	split := strings.Split(p, "/")
	if len(split) <= 1 {
		http.Error(resp, "Requests for the /golink endpoint should look like /golink/<name>.", http.StatusBadRequest)
		return
	}
	name := split[1]
	link, ok, err := gl.linkByName(ctx, name)
	if err != nil {
		log.Printf("Query for name=%q failed: %v", name, err)
		http.Error(resp, "Failed to lookup %q in the database.", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.NotFound(resp, req)
		return
	}
	if err := goLinkPageTemplate.Execute(resp, link); err != nil {
		log.Printf("Failed to execute golink page template for %q: %v", link, err)
		http.Error(resp, "Failed to create a page for the golink.", http.StatusInternalServerError)
		return
	}
}

func (gl *GoLink) updateHandler(resp http.ResponseWriter, req *http.Request) {}

func (gl *GoLink) deleteHandler(resp http.ResponseWriter, req *http.Request) {}

func (gl *GoLink) goHandler(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	p := req.URL.EscapedPath()
	p = strings.TrimPrefix(p, "/")
	split := strings.Split(p, "/")
	if len(split) <= 1 {
		http.Error(resp, "Requests for the /go endpoint should look like /go/<name>.", http.StatusBadRequest)
		return
	}
	name := split[1]
	link, ok, err := gl.linkByName(ctx, name)
	if err != nil {
		log.Printf("Failed to lookup name=%q: %v", name, err)
		http.Error(resp, "Failed to lookup name %q.", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.NotFound(resp, req)
		return
	}
	http.Redirect(resp, req, link.Link, http.StatusTemporaryRedirect)
}

func (gl *GoLink) linkByName(ctx context.Context, name string) (*golink, bool, error) {
	const query = "select (url) from links where name=?;"
	row := gl.db.QueryRowContext(ctx, query, name)
	var link string
	if err := row.Scan(&link); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &golink{name, link}, true, nil
}
