package main

import (
	"dnd/dice"
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

type DiceServer struct {
	Template       *template.Template
	PreviousRolls  []dice.RollResult
	LastCustomRoll string
}

type RollTemplateValues struct {
	HasResult      bool
	LastRoll       *dice.RollResult
	OlderRolls     chan *dice.RollResult
	LastCustomRoll *string
}

func (diceServer *DiceServer) HandleGet(w http.ResponseWriter, r *http.Request) {
	var templateValues RollTemplateValues
	templateValues.LastCustomRoll = &diceServer.LastCustomRoll
	if len(diceServer.PreviousRolls) > 0 {
		templateValues.HasResult = true
		penultimateIndex := len(diceServer.PreviousRolls) - 1
		templateValues.LastRoll = &diceServer.PreviousRolls[penultimateIndex]
		templateValues.OlderRolls = dice.ReverseRollResultSlice(
			diceServer.PreviousRolls[:penultimateIndex])
	}
	err := diceServer.Template.Execute(w, templateValues)
	if err != nil {
		log.Print(err)
	}
}

func (diceServer *DiceServer) HandlePost(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	roll, err := dice.ParseRollString(r.Form["roll"][0])
	if len(r.Form["roll-custom"]) > 0 {
		diceServer.LastCustomRoll = r.Form["roll"][0]
	}
	if err == nil {
		diceServer.PreviousRolls = append(diceServer.PreviousRolls, roll.Simulate())
	} else {
		log.Println(err)
	}
	http.Redirect(w, r, "/", 303)
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	theTemplate, err := template.ParseGlob("templates/*")
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Starting dice server on localhost:1212...")
	diceServer := DiceServer{
		theTemplate.Lookup("roll.html.tmpl"),
		make([]dice.RollResult, 0),
		"Custom Roll"}
	encounterServer := EncounterServer{
		theTemplate.Lookup("encounter.html.tmpl"),
		nil}

	// This is a bit messy to always get the same static handling, but stateful otherwise.
	// Its possible all this chaining and calls is inefficient, but it shouldn't be a very hot path.
	// Maybe writing a local application while using exclusively server-side logic is an unusual,
	// bad, and not-well-supported pattern. Who knew?
	server := http.NewServeMux()
	server.HandleFunc("/favicon.ico", http.NotFound)
	server.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("static"))))

	initialisationHandler, err := newInitialisationServer(getDataDir(), theTemplate.Lookup("choosegroup.html.tmpl"))
	if err != nil {
		log.Fatalf("Catacylsmic error initialising - %v", err)
	}

	logicServer := http.NewServeMux()
	logicServer.Handle("/encounter/", StandardGetPostHandler(&encounterServer))
	logicServer.Handle("/", StandardGetPostHandler(&diceServer))

	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if initialisationHandler.InitialisationComplete {
			logicServer.ServeHTTP(w, r)
		} else {
			StandardGetPostHandler(initialisationHandler).ServeHTTP(w, r)
			if initialisationHandler.InitialisationComplete {
				encounterServer.party = initialisationHandler.Party
			}
		}
	})
	err = http.ListenAndServe("localhost:1212", server)
	if err != nil {
		log.Fatal(err)
	}
}
