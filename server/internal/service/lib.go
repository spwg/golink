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

type golink struct {
	Name string
	Link string
}

// New creates a *GoLink.
func New(db *sql.DB) *GoLink {
	return &GoLink{db}
}

// Run installs and starts up the service.
func (gl *GoLink) Run(ctx context.Context, port int) error {
	if err := gl.startUp(ctx, port); err != nil {
		return err
	}
	return nil
}

func (gl *GoLink) startUp(ctx context.Context, port int) error {
	httpAddr := fmt.Sprintf("127.0.0.1:%v", port)
	log.Printf("Server listening on http://%v", httpAddr)
	mux := http.NewServeMux()
	mux.HandleFunc("/", gl.indexHandler)
	mux.HandleFunc("/create_golink", gl.createHandler)
	mux.HandleFunc("/golink/", gl.readHandler)
	mux.HandleFunc("/update_golink", gl.updateHandler)
	mux.HandleFunc("/delete_golink", gl.deleteHandler)
	mux.HandleFunc("/go", gl.goHandler)
	mux.HandleFunc("/go/", gl.goHandler)
	server := &http.Server{
		Addr:    httpAddr,
		Handler: gl.logRequestHandler(mux),
	}
	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("listen and serve failed: %v", err)
	}
	return nil
}

func (gl *GoLink) logRequestHandler(h http.Handler) http.Handler {
	fn := func(resp http.ResponseWriter, req *http.Request) {
		log.Printf("%+v", req)
		h.ServeHTTP(resp, req)
	}
	return http.HandlerFunc(fn)
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
			http.Redirect(resp, req, link.Link, http.StatusTemporaryRedirect)
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
	if !validLinkName(name) {
		http.Error(resp, fmt.Sprintf("The golink's name cannot contain any characters like %q.", blockChars), http.StatusBadRequest)
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
	if !validLinkName(reqName) {
		http.Error(resp, fmt.Sprintf("The golink's name cannot contain any characters like %q.", blockChars), http.StatusBadRequest)
		return
	}
	reqLink := req.PostForm.Get("link")
	if reqLink == "" {
		http.Error(resp, "Invalid form: missing the link.", http.StatusBadRequest)
		return
	}
	if _, found, err := gl.linkByName(ctx, oldName); err != nil {
		log.Printf("Failed to lookup name=%q: %v", oldName, err)
		http.Error(resp, fmt.Sprintf("Failed to lookup link %q.", oldName), http.StatusInternalServerError)
		return
	} else if found {
		http.Error(resp, fmt.Sprintf("Link for %q already exists.", oldName), http.StatusBadRequest)
		return
	}
	// There is a race here between checking that the new name doesn't exist the
	// update, but the checks are really just for writing nicer messages for the
	// user. The database will enforce that names are unique as a constraint.
	if _, found, err := gl.linkByName(ctx, reqName); err != nil {
		log.Printf("Failed to lookup name=%q: %v", reqName, err)
		http.Error(resp, fmt.Sprintf("Failed to lookup link %q.", reqName), http.StatusInternalServerError)
		return
	} else if found {
		http.Error(resp, fmt.Sprintf("Link for %q already exists.", reqName), http.StatusBadRequest)
		return
	}
	const query = "update links set name = ?, url = ? where name = ?;"
	if _, err := gl.db.ExecContext(ctx, query, reqName, reqLink, oldName); err != nil {
		log.Printf("Failed to update link name=%q to name=%q link=%q: %v", oldName, reqName, reqLink, err)
		http.Error(resp, fmt.Sprintf("Failed to update link name=%q", oldName), http.StatusBadRequest)
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

func validLinkName(name string) bool {
	return !strings.Contains(name, blockChars)
}
