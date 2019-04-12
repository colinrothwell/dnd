package main

import (
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func getURLFunctionAndArgument(u *url.URL) (function string, argument string) {
	urlParts := strings.Split(u.Path, "/")
	if len(urlParts) == 0 {
		return "", ""
	} else if len(urlParts) == 1 {
		return "", urlParts[0]
	} else {
		return strings.Join(urlParts[:len(urlParts)-1], "/"), urlParts[len(urlParts)-1]
	}
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
			err := t.GetTemplate().Execute(w, data)
			if err != nil {
				log.Print(err)
			}
		}
	})
}

type DndServer struct {
	DiceServer      DiceServer
	EncounterServer EncounterServer
	StaticServer    http.Handler
}

func loadTemplate(name string) *template.Template {
	// This rigamorale implements template inheritance. frame is the template we want to execute
	// but with different templates defined from HeadContent and BodyContent.
	t := template.New("frame.html.tmpl")
	// This is clumsy, but is to set empty default implementations
	t, err := t.Parse(`{{define "HeadContent"}}{{end}}{{define "BodyContent"}}{{end}}`)
	if err != nil {
		log.Fatalf("Error loading template '%s' - %v", name, err)
	}
	t, err = t.ParseFiles("templates/frame.html.tmpl", "templates/"+name+".tmpl")
	if err != nil {
		log.Fatalf("Error loading template '%s' - %v", name, err)
	}
	return t
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	initialisationHandler, err := newInitialisationServer(getDataDir(), loadTemplate("choosegroup.html"))
	if err != nil {
		log.Fatalf("Catacylsmic error initialising - %v", err)
	}

	logicServer := http.NewServeMux()
	server := http.NewServeMux()
	server.HandleFunc("/favicon.ico", http.NotFound)
	server.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("static"))))
	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if initialisationHandler.InitialisationComplete {
			logicServer.ServeHTTP(w, r)
		} else {
			StandardGetPostHandler(initialisationHandler).ServeHTTP(w, r)
			if initialisationHandler.InitialisationComplete {
				diceServer := DiceServer{
					loadTemplate("roll.html"),
					initialisationHandler.Party}
				encounterServer := EncounterServer{
					loadTemplate("encounter.html"),
					initialisationHandler.Party}
				logicServer.Handle("/encounter", StandardTemplatedGetPostHandler(&encounterServer))
				logicServer.Handle("/roll", StandardTemplatedGetPostHandler(&diceServer))
			}
		}
	})
	log.Print("Starting encounter server on localhost:1212...")
	err = http.ListenAndServe("localhost:1212", server)
	if err != nil {
		log.Fatal(err)
	}
}
