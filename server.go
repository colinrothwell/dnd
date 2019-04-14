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

type GetPostHandler interface {
	HandleGet(w http.ResponseWriter, r *http.Request)
	HandlePost(w http.ResponseWriter, r *http.Request)
}

func StandardGetPostHandler(h GetPostHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			h.HandlePost(w, r)
		} else {
			h.HandleGet(w, r)
		}
	})
}

// A TemplatedGetPostHandler renders get requests using a template
type TemplatedGetPostHandler interface {
	GetTemplate() *template.Template
	GenerateTemplateData(r *http.Request) interface{}
	HandlePost(w http.ResponseWriter, r *http.Request)
}

// StandardTemplatedGetPostHandler converts a TemplatedGetPostHandler to an http.Handler
func StandardTemplatedGetPostHandler(t TemplatedGetPostHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			t.HandlePost(w, r)
		} else {
			data := t.GenerateTemplateData(r)
			temp := t.GetTemplate()
			temp = temp.Funcs(template.FuncMap{"redirectURIInput": func() template.HTML {
				input := "<input type=\"hidden\" name=\"redirectURI\" value=\"" + r.RequestURI + "\" />"
				return template.HTML(input)
			}})
			err := temp.Execute(w, data)
			if err != nil {
				log.Print(err)
			}
		}
	})
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
	initialisationHandler := StandardTemplatedGetPostHandler(initialisationServer)

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
				encounterServer := EncounterServer{encounterTemplate, initialisationServer.Party}
				overviewServer := CreateOverviewServer(overviewTemplate, &encounterServer, &diceServer)
				logicServer.Handle("/encounter/", StandardTemplatedGetPostHandler(&encounterServer))
				logicServer.Handle("/roll/", StandardTemplatedGetPostHandler(&diceServer))
				logicServer.Handle("/", StandardTemplatedGetPostHandler(overviewServer))
			}
		}
	})
	log.Print("Starting encounter server on localhost:1212...")
	err = http.ListenAndServe("localhost:1212", server)
	if err != nil {
		log.Fatal(err)
	}
}
