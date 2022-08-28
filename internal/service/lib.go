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
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/spwg/golink/internal/link"
)

const (
	indexPage = `<html>
<body>
<p><b>Create a new link</b></p>
<form action="/create_golink" method="post">
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
<p><b>Manage golink</b></p>
<p>Name: {{.Name}}</p>
<p>URL: <a href="/go/{{.Name}}">{{.Link}}</a></p>
<p><b>Change golink</b></p>
<form action="/update_golink" method="post">
<label for="name">Link name:</label><br>
<input required type="text" id="name" value={{.Name}} name="name"><br>
<label for="link">Link</label><br>
<input required type="url" id="link" value={{.Link}} name="link"><br>
<input hidden type="text" id="old_name" name="old_name" value="{{.Name}}">
<input type="submit" value="Change">
</form>
<p><b>Delete golink</b></p>
<form action="/delete_golink" method="post">
<input hidden type="text" id="name" value={{.Name}} name="name">
<input type="submit", value="Delete">
</form>
</body>
</html>`
	blockChars = "/<>"
)

var (
	indexPageTemplate  = template.Must(template.New("").Parse(indexPage))
	goLinkPageTemplate = template.Must(template.New("").Parse(goLinkPage))
)

// GoLink is a service for shortened links.
type GoLink struct {
	db *sql.DB
}

// New creates a *GoLink.
func New(db *sql.DB) *GoLink {
	return &GoLink{db}
}

// Run installs and starts up the service.
func (gl *GoLink) Run(ctx context.Context, addr string) error {
	if err := gl.startUp(ctx, addr); err != nil {
		return err
	}
	return nil
}

func (gl *GoLink) startUp(ctx context.Context, addr string) error {
	log.Printf("Server listening on %s", addr)
	mux := http.NewServeMux()
	mux.HandleFunc("/", gl.indexHandler)
	mux.HandleFunc("/favicon.ico", gl.faviconHandler)
	mux.HandleFunc("/create_golink", gl.createHandler)
	mux.HandleFunc("/golink/", gl.readHandler)
	mux.HandleFunc("/update_golink", gl.updateHandler)
	mux.HandleFunc("/delete_golink", gl.deleteHandler)
	mux.HandleFunc("/go", gl.goHandler)
	mux.HandleFunc("/go/", gl.goHandler)
	server := &http.Server{
		Addr:    addr,
		Handler: logHandler(mux),
	}
	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("listen and serve failed: %v", err)
	}
	return nil
}

func logHandler(h http.Handler) http.Handler {
	fn := func(resp http.ResponseWriter, req *http.Request) {
		b, err := httputil.DumpRequest(req, true)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("%s\n", b)
		recorder := httptest.NewRecorder()
		h.ServeHTTP(resp, req)
		b, err = httputil.DumpResponse(recorder.Result(), true)
		if err != nil {
			log.Printf("Failed to log http request: %v", err)
			return
		}
		log.Printf("%s\n", b)
	}
	return http.HandlerFunc(fn)
}

func (gl *GoLink) faviconHandler(resp http.ResponseWriter, req *http.Request) {
	http.NotFound(resp, req)
}

func (gl *GoLink) indexHandler(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	p := req.URL.EscapedPath()
	p = strings.TrimPrefix(p, "/")
	if p != "" {
		// Requests for go/name will map to p == "name" here, so we need to redirect.
		link, found, err := gl.linkByName(ctx, p)
		if err != nil {
			log.Printf("Failed to lookup %q: %v", p, err)
			http.Error(resp, fmt.Sprintf("Failed to lookup %q.", p), http.StatusInternalServerError)
			return
		}
		if found {
			log.Printf("Redirecting %q -> %q", req.URL.String(), link.Link.String())
			http.Redirect(resp, req, link.Link.String(), http.StatusTemporaryRedirect)
			return
		}
		if !found {
			http.NotFound(resp, req)
			return
		}
	}
	const query = "select name, url from links;"
	rows, err := gl.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Failed to query all links in the database: %v", err)
		http.Error(resp, "Failed to query all links in the database.", http.StatusInternalServerError)
		return
	}
	var links []*link.Record
	for rows.Next() {
		var name, address string
		if err := rows.Scan(&name, &address); err != nil {
			log.Printf("Failed to scan link: %v", err)
			http.Error(resp, "Failed to query all links in the databse.", http.StatusInternalServerError)
			return
		}
		u, err := url.Parse(address)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}
		links = append(links, &link.Record{Name: name, Link: u})
	}
	if err := indexPageTemplate.Execute(resp, struct {
		Links []*link.Record
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
	l := req.PostForm.Get("link")
	err := link.Create(ctx, gl.db, name, l)
	if err != nil {
		switch err {
		case link.ErrAlreadyExists:
			msg := fmt.Sprintf("The golink %q already exists.", name)
			http.Error(resp, msg, http.StatusConflict)
			return
		case link.ErrInvalidLinkName:
			msg := fmt.Sprintf(`Invalid link name. Must not be "" or contain %q.`, link.BlockChars)
			http.Error(resp, msg, http.StatusBadRequest)
			return
		case link.ErrUnparseableAddress:
			msg := fmt.Sprintf("Invalid URL %q: not parseable.", l)
			http.Error(resp, msg, http.StatusBadRequest)
			return
		}
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Saved new link: %v -> %v", name, l)
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
	record, err := link.Read(ctx, gl.db, name)
	if err != nil {
		switch err {
		case link.ErrNotFound:
			http.NotFound(resp, req)
			return
		}
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := goLinkPageTemplate.Execute(resp, record); err != nil {
		log.Printf("Failed to execute golink page template for %v: %v", record, err)
		http.Error(resp, "Failed to create a page for the golink.", http.StatusInternalServerError)
		return
	}
}

func (gl *GoLink) updateHandler(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	if err := req.ParseForm(); err != nil {
		log.Printf("Failed to parse form: %v", err)
		http.Error(resp, "Failed to parse form.", http.StatusBadRequest)
		return
	}
	oldName := req.PostForm.Get("old_name")
	if oldName == "" {
		http.Error(resp, "Invalid form: missing the old name of the link.", http.StatusBadRequest)
		return
	}
	reqName := req.PostForm.Get("name")
	if reqName == "" {
		http.Error(resp, "Invalid form: missing the new name of the link.", http.StatusBadRequest)
		return
	}
	reqLink := req.PostForm.Get("link")
	if reqLink == "" {
		http.Error(resp, "Invalid form: missing the link.", http.StatusBadRequest)
		return
	}
	if err := link.Update(ctx, gl.db, oldName, reqName, reqLink); err != nil {
		switch err {
		case link.ErrAlreadyExists:
			msg := fmt.Sprintf("Link for %q already exists.", reqName)
			http.Error(resp, msg, http.StatusBadRequest)
			return
		case link.ErrInvalidLinkName:
			msg := fmt.Sprintf(`Invalid link name %q. Must not be "" or contain %q.`, reqName, link.BlockChars)
			http.Error(resp, msg, http.StatusBadRequest)
			return
		case link.ErrUnparseableAddress:
			msg := fmt.Sprintf("Invalid address %q: failed to parse.", reqLink)
			http.Error(resp, msg, http.StatusBadRequest)
			return
		}
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(resp, req, "/golink/"+reqName, http.StatusTemporaryRedirect)
}

func (gl *GoLink) deleteHandler(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	if req.Method == "GET" {
		http.Error(resp, "GET method not supported.", http.StatusMethodNotAllowed)
		return
	}
	if err := req.ParseForm(); err != nil {
		http.Error(resp, "Failed to parse form.", http.StatusBadRequest)
		return
	}
	name := req.PostForm.Get("name")
	_, found, err := gl.linkByName(ctx, name)
	if err != nil {
		log.Printf("Failed to lookup name=%q: %v", name, err)
		http.Error(resp, fmt.Sprintf("Failed to lookup %q.", name), http.StatusInternalServerError)
		return
	}
	if !found {
		http.NotFound(resp, req)
		return
	}
	const query = "delete from links where name=?;"
	if _, err := gl.db.ExecContext(ctx, query, name); err != nil {
		log.Printf("Failed to delete name=%q: %v", name, err)
		http.Error(resp, "Failed to delete %q.", http.StatusInternalServerError)
		return
	}
	http.Redirect(resp, req, "/", http.StatusTemporaryRedirect)
}

func (gl *GoLink) goHandler(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	p := req.URL.EscapedPath()
	if p == "/go" || p == "/go/" {
		http.Redirect(resp, req, "/", http.StatusTemporaryRedirect)
		return
	}
	p = strings.TrimPrefix(p, "/")
	split := strings.Split(p, "/")
	if len(split) <= 1 {
		http.Error(resp, "Requests for the /go endpoint should look like /go/<name>.", http.StatusBadRequest)
		return
	}
	name := split[1]
	l, ok, err := gl.linkByName(ctx, name)
	if err != nil {
		log.Printf("Failed to lookup name=%q: %v", name, err)
		http.Error(resp, "Failed to lookup name %q.", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.NotFound(resp, req)
		return
	}
	http.Redirect(resp, req, l.Link.String(), http.StatusTemporaryRedirect)
}

func (gl *GoLink) linkByName(ctx context.Context, name string) (*link.Record, bool, error) {
	const query = "select (url) from links where name=?;"
	row := gl.db.QueryRowContext(ctx, query, name)
	var s string
	if err := row.Scan(&s); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, false, err
	}
	return &link.Record{Name: name, Link: u}, true, nil
}
