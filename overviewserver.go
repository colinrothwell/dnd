package main

import (
	"dnd/party"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

type OverviewServer struct {
	template        *template.Template
	encounterServer *EncounterServer
	diceServer      *DiceServer
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

func NewOverviewServer(t *template.Template, es *EncounterServer, ds *DiceServer) *OverviewServer {
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
	return &OverviewServer{t, es, ds}
}

func (os *OverviewServer) GetTemplate() *template.Template {
	return os.template
}

type OverviewTemplateData struct {
	EncounterData interface{}
	DiceData      interface{}
}

func (os *OverviewServer) GenerateTemplateData(r *http.Request, p party.Party) interface{} {
	data := OverviewTemplateData{
		os.encounterServer.GenerateTemplateData(r, p),
		os.diceServer.GenerateTemplateData(r),
	}
	return data
}

func (os *OverviewServer) HandlePost(r *http.Request) error {
	return fmt.Errorf("Error! OverviewServer should never get post requests")
}
