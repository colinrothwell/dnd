package main

import (
	"dnd/party"
	"errors"
	"html/template"
	"net/http"
	"strconv"
)

type InitiativeServer struct {
	entryTemplate *template.Template
}

// GetTemplate gets the encounter or view template as appropriate
func (s *InitiativeServer) GetTemplate() *template.Template {
	return s.entryTemplate
}

type creatureInitiativeInformation struct {
	Name, InputName, Value string
}

type initiativeTemplateData struct {
	PlayerInformation []*creatureInitiativeInformation
}

// GenerateTemplateData returns the data for the template
func (s *InitiativeServer) GenerateTemplateData(r *http.Request, p party.Party) interface{} {
	pis := p.PlayerInitiatives()
	data := &initiativeTemplateData{
		make([]*creatureInitiativeInformation, len(pis))}
	for i, pi := range pis {
		initiativeString := ""
		if pi.HasInitiative {
			initiativeString = strconv.Itoa(pi.Initiative)
		}
		data.PlayerInformation[i] = &creatureInitiativeInformation{
			pi.Name,
			strconv.Itoa(i),
			initiativeString}
	}
	return data
}

// HandlePost needs to work out whether to roll the initiatives or add a new
// creature or party member
func (s *InitiativeServer) HandlePost(r *http.Request, p party.Party) (party.ReversibleAction, error) {
	return nil, errors.New("unimplemented :(")

}
