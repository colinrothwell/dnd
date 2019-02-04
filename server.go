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

type DndServer struct {
	DiceServer      DiceServer
	EncounterServer EncounterServer
	StaticServer    http.Handler
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	theTemplate, err := template.ParseGlob("templates/*")
	if err != nil {
		log.Fatal(err)
	}

	initialisationHandler, err := newInitialisationServer(getDataDir(), theTemplate.Lookup("choosegroup.html.tmpl"))
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
					theTemplate.Lookup("roll.html.tmpl"),
					initialisationHandler.Party}
				encounterServer := EncounterServer{
					theTemplate.Lookup("encounter.html.tmpl"),
					initialisationHandler.Party}
				logicServer.Handle("/encounter/", StandardGetPostHandler(&encounterServer))
				logicServer.Handle("/", StandardGetPostHandler(&diceServer))
			}
		}
	})
	log.Print("Starting encounter server on localhost:1212...")
	err = http.ListenAndServe("localhost:1212", server)
	if err != nil {
		log.Fatal(err)
	}
}
