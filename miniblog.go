package main

import (
	"embed"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
)

var (
	//go:embed templates/edit.html templates/view.html
	res   embed.FS
	pages = map[string]string{
		"/view/": "view.html",
		"/edit/": "edit.html",
	}
	templates = template.Must(template.ParseFS(res, "templates/view.html", "templates/edit.html"))
	validPath = regexp.MustCompile("^/(edit|save|view|view500)/([a-zA-Z0-9]+)$")
)

func main() {
	http.HandleFunc("/view/", makeHandler(handleView))
	http.HandleFunc("/view500/", makeHandler(handleView500))
	http.HandleFunc("/edit/", makeHandler(handleEdit))
	http.HandleFunc("/save/", makeHandler(handleSave))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func makeHandler(fn func(w http.ResponseWriter, r *http.Request, title string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// extract title and call fn with it
		title, err := getTitle(r.URL.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, title)
	}
}

type Page struct {
	Title string
	Body  []byte
}

func (p *Page) save() error {
	filename := p.Title + ".txt"
	return os.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func handleView(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, pages["/view/"], p)
}

func handleView500(w http.ResponseWriter, _ *http.Request, _ string) {
	log.Printf("Returning error 500")
	http.Error(w, "500", http.StatusInternalServerError)
}

func handleEdit(w http.ResponseWriter, _ *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, pages["/edit/"], p)
}

func handleSave(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	if err := p.save(); err != nil {
		log.Printf("error saving page %q: %s", p.Title, err)
		http.Redirect(w, r, "/edit/"+title, http.StatusNotModified)
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl, p)
	if err != nil {
		log.Printf("error executing template %s: %s", tmpl, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getTitle(urlPath string) (string, error) {
	m := validPath.FindStringSubmatch(urlPath)
	if m == nil {
		return "", errors.New("invalid page title")
	}
	return m[2], nil
}
