package main

import (
	"dnd/party"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

type PartyInitialisationData struct {
	Name, URL string
}

type initialisationServer struct {
	dataDir                string
	template               *template.Template
	parties                []party.Party
	InitialisationComplete bool
	Party                  party.Party
}

func getDataDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	dataDir := filepath.Join(usr.HomeDir, "encounters")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		log.Printf("encounters directory '%s' does not exist, creating...", dataDir)
		err = os.Mkdir(dataDir, 0750)
		if err != nil {
			log.Printf("Oh no, I don't believe it! Error creating directory - %v", err)
		}
	}
	return dataDir
}

func newInitialisationServer(dataDir string, t *template.Template) (*initialisationServer, error) {
	s := initialisationServer{dataDir, t, make([]party.Party, 0), false, nil}
	contents, err := ioutil.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("error when reading data directory - %v", err)
	}
	i := 0
	for _, fileInfo := range contents {
		if !fileInfo.IsDir() && strings.HasSuffix(fileInfo.Name(), ".party.gob") {
			fileName := filepath.Join(s.dataDir, fileInfo.Name())
			file, err := os.Open(fileName)
			if err != nil {
				log.Printf("Error opening .party.gob file - %v", err)
			}
			defer file.Close()
			p, err := party.Load(file)
			if err != nil {
				log.Printf("Error loading party '%s' - %v", fileName, err)
				continue
			}
			s.parties = append(s.parties, p)
			i++
		}
	}
	return &s, nil
}

func (s *initialisationServer) GetTemplate() *template.Template {
	return s.template
}

func (s *initialisationServer) GenerateTemplateData(r *http.Request) interface{} {
	partyInitialisationData := make([]PartyInitialisationData, len(s.parties))
	for i, p := range s.parties {
		partyInitialisationData[i] = PartyInitialisationData{
			p.Name(),
			fmt.Sprintf("/%d", i)}
	}
	return partyInitialisationData
}

func (s *initialisationServer) initialiseWithNewParty(name string) error {
	s.Party = party.New(s.dataDir, name)
	err := s.Party.Save()
	if err != nil {
		log.Printf("Error encoding party - %v", err)
	}
	log.Printf("Created new party '%s'", name)
	return nil
}

func (s *initialisationServer) HandlePost(r *http.Request) error {
	argument, err := getURLArgument(r.URL)
	if err != nil {
		return fmt.Errorf("error getting argument - %v", err)
	}
	if argument == "" {
		err := s.initialiseWithNewParty(r.Form["partyName"][0])
		if err != nil {
			return fmt.Errorf("error creating file for party - %v", err)
		}
	} else {
		i, err := strconv.Atoi(argument)
		if err != nil {
			return fmt.Errorf("error parsing party index - %v", err)
		}
		if i < 0 || i >= len(s.parties) {
			return fmt.Errorf("party index %d out of range", i)
		}
		s.Party = s.parties[i]
	}
	s.InitialisationComplete = true
	return nil
}
