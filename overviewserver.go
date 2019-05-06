package main

import (
	"dnd/party"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

type OverviewServer struct {
	template         *template.Template
	encounterServer  *EncounterServer
	diceServer       *DiceServer
	initiativeServer *InitiativeServer
}

// This is a bit complicated. I've implemented this slightly crazy template inheritance system.
// In order to do this overview pages, I want to render a bunch of different BodyContent templates
// from other templates. To do this, I have to prefix them with a source, and then preserve the
// original root of the template.
func attachPrefixedTemplate(root, child *template.Template, prefix string) (*template.Template, error) {
	rootName := root.Name()
	t, err := root.AddParseTree(prefix+child.Name(), child.Tree)
	if err != nil {
		return nil, err
	}
	return t.Lookup(rootName), nil
}

// NewOverviewServer creeates a new overview server attaching the templates that won't change
// directly. We regenerate in GetTemplate() because InitiativeServer changes its template
// based on if it is running or being edited
func NewOverviewServer(t *template.Template, es *EncounterServer,
	ds *DiceServer, is *InitiativeServer) *OverviewServer {

	t, err := attachPrefixedTemplate(t, es.GetTemplate().Lookup("BodyContent"), "Encounter")
	if err != nil {
		log.Fatalf("Catastrophic error attaching encounter body content - %v", err)
	}
	t, err = attachPrefixedTemplate(t, ds.GetTemplate().Lookup("BodyContent"), "Roll")
	if err != nil {
		log.Fatalf("Catastrophic error attaching roll body content - %v", err)
	}
	t, err = attachPrefixedTemplate(t, ds.GetTemplate().Lookup("HeadContent"), "Roll")
	if err != nil {
		log.Fatalf("Catastrophic error attaching roll head content - %v", err)
	}
	return &OverviewServer{t, es, ds, is}
}

func (os *OverviewServer) GetTemplate() *template.Template {
	t, err := attachPrefixedTemplate(os.template,
		os.initiativeServer.GetTemplate().Lookup("BodyContent"), "Initiative")
	if err != nil {
		log.Fatalf("Catastrophic error attaching initiative head content: %v", err)
	}
	return t
}

type OverviewTemplateData struct {
	InitiativeData interface{}
	EncounterData  interface{}
	DiceData       interface{}
	UndoDisabled   string
	RedoDisabled   string
}

func (os *OverviewServer) GenerateTemplateData(r *http.Request, p party.Party) interface{} {
	var undoDisabled, redoDisabled string
	if !p.CanUndo() {
		undoDisabled = "disabled"
	}
	if !p.CanRedo() {
		redoDisabled = "disabled"
	}
	data := OverviewTemplateData{
		os.initiativeServer.GenerateTemplateData(r, p),
		os.encounterServer.GenerateTemplateData(r, p),
		os.diceServer.GenerateTemplateData(r),
		undoDisabled,
		redoDisabled,
	}
	return data
}

// HandlePost handles undoing and redoing. Always returns action nil because it is a bit special
// and operates outside the usual action flow.
func (os *OverviewServer) HandlePost(r *http.Request, p party.Party) (party.ReversibleAction, error) {
	switch r.URL.Path {
	case "/undo":
		err := p.Undo()
		if err != nil {
			return nil, fmt.Errorf("error attempting undo: %v", err)
		}
	case "/redo":
		err := p.Redo()
		if err != nil {
			return nil, fmt.Errorf("error attempting redo: %v", err)
		}
	default:
		return nil, fmt.Errorf("unrecognised endpoint: '%v'", r.URL.Path)
	}
	return nil, nil
}
