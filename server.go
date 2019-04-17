package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func getURLArgument(u *url.URL) (string, error) {
	slashLastIndex := strings.LastIndex(u.Path, "/")
	if slashLastIndex == -1 {
		return "", fmt.Errorf("no / in url path %s", u.Path)
	}
	return u.Path[slashLastIndex+1:], nil
}

func sumIntSlice(slice []int) int {
	result := 0
	for _, n := range slice {
		result += n
	}
	return result
}

type getPostHandler struct {
	get, post http.Handler
}

func (h *getPostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		h.post.ServeHTTP(w, r)
	} else {
		h.get.ServeHTTP(w, r)
	}
}

// A TemplatedGetHandler renders get requests using a template
type TemplatedGetHandler interface {
	GetTemplate() *template.Template
	GenerateTemplateData(*http.Request) interface{}
}

type standardTemplatedGetHandler struct {
	TemplatedGetHandler
}

func (h *standardTemplatedGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := h.GenerateTemplateData(r)
	temp := h.GetTemplate()
	temp = temp.Funcs(template.FuncMap{"redirectURIInput": func() template.HTML {
		input := "<input type=\"hidden\" name=\"redirectURI\" value=\"" + r.RequestURI + "\" />"
		return template.HTML(input)
	}})
	err := temp.Execute(w, data)
	if err != nil {
		log.Print(err)
	}
}

type TemplatedGetPostHandler interface {
	TemplatedGetHandler
	HandlePost(http.ResponseWriter, *http.Request)
}

func standardTemplatedGetPostHandler(h TemplatedGetPostHandler) http.Handler {
	return &getPostHandler{
		&standardTemplatedGetHandler{h},
		http.HandlerFunc(h.HandlePost)}
}

// ParseFormAndGetRedirectURI parses the form associated with an HTTP request, and returns the URI
// to redirect to after finishing handling the request. It returns "/" in case of an error.
func ParseFormAndGetRedirectURI(r *http.Request) (string, error) {
	err := r.ParseForm()
	if err != nil {
		return "/", err
	}
	if redirectURISlice, ok := r.Form["redirectURI"]; ok {
		if len(redirectURISlice) != 1 {
			return "/", fmt.Errorf("multiple possible candidates for redirectURI - %v", redirectURISlice)
		}
		return redirectURISlice[0], nil
	}
	return "/", fmt.Errorf("form '%v' didn't contain redirectURI", r.Form)
}

type RedirectPostHandler interface {
	HandlePost(*http.Request) error
}

type standardRedirectPostHandler struct {
	RedirectPostHandler
}

func (h *standardRedirectPostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	redirectURI, err := ParseFormAndGetRedirectURI(r)
	if err != nil {
		log.Printf("Error getting direct URL from request to '%v'", r.RequestURI)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	err = h.HandlePost(r)
	if err != nil {
		log.Printf("Error handling post - %v", err)
	}
	http.Redirect(w, r, redirectURI, http.StatusSeeOther)
}

// TemplatedGetRedirectPostHandler does what it says on the tin: this is the pattern
// encounter follows.
type TemplatedGetRedirectPostHandler interface {
	TemplatedGetHandler
	RedirectPostHandler
}

func standardTemplatedGetRedirectPostHandler(h TemplatedGetRedirectPostHandler) http.Handler {
	return &getPostHandler{
		&standardTemplatedGetHandler{h},
		&standardRedirectPostHandler{h}}
}

func loadTemplate(name string) *template.Template {
	// This rigamorale implements template inheritance. frame is the template we want to execute
	// but with different templates defined from HeadContent and BodyContent.
	t := template.New("frame.html.tmpl")
	// This gets overwritten at the call site, but we can't parse a template with a missing function
	// for some reason. This is the simplest func that works nil or no results don't
	t = t.Funcs(template.FuncMap{"redirectURIInput": func() string { return "DEADBEEF" }})
	// This is clumsy, but is to set empty default implementations
	t, err := t.Parse(`{{define "HeadContent"}}{{end}}{{define "BodyContent"}}{{end}}`)
	if err != nil {
		log.Fatalf("Error parsing empty content templates '%s' - %v", name, err)
	}
	t, err = t.ParseFiles("templates/frame.html.tmpl", "templates/"+name+".tmpl")
	if err != nil {
		log.Fatalf("Error loading templates from file '%s' - %v", name, err)
	}
	return t
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	initialisationServer, err := newInitialisationServer(getDataDir(), loadTemplate("choosegroup.html"))
	if err != nil {
		log.Fatalf("Catacylsmic error initialising - %v", err)
	}
	initialisationHandler := standardTemplatedGetRedirectPostHandler(initialisationServer)

	diceTemplate := loadTemplate("roll.html")
	encounterTemplate := loadTemplate("encounter.html")
	overviewTemplate := loadTemplate("overview.html")

	logicServer := http.NewServeMux()
	server := http.NewServeMux()
	server.HandleFunc("/favicon.ico", http.NotFound)
	server.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("static"))))
	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if initialisationServer.InitialisationComplete {
			logicServer.ServeHTTP(w, r)
		} else {
			initialisationHandler.ServeHTTP(w, r)
			if initialisationServer.InitialisationComplete {
				diceServer := DiceServer{diceTemplate, initialisationServer.Party}
				encounterServer, err := NewEncounterServer(encounterTemplate, initialisationServer.Party)
				if err != nil {
					log.Fatalf("Couldn't create encounter server - %v", err)
				}
				overviewServer := NewOverviewServer(overviewTemplate, encounterServer, &diceServer)
				logicServer.Handle("/encounter/", standardTemplatedGetRedirectPostHandler(encounterServer))
				logicServer.Handle("/roll/", standardTemplatedGetPostHandler(&diceServer))
				logicServer.Handle("/", standardTemplatedGetPostHandler(overviewServer))
			}
		}
	})
	log.Print("Starting encounter server on localhost:1212...")
	err = http.ListenAndServe("localhost:1212", server)
	if err != nil {
		log.Fatal(err)
	}
}
