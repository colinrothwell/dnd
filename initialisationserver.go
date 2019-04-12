package main

import (
	"encoding/gob"
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
	parties                []Party
	InitialisationComplete bool
	Party                  *Party
}

func getDataDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	dataDir := filepath.Join(usr.HomeDir, "encounters")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		log.Printf("encounters directory '%s' does not exist, creating...", dataDir)
		os.Mkdir(dataDir, 0755)
	}
	return dataDir
}

func newInitialisationServer(dataDir string, t *template.Template) (*initialisationServer, error) {
	s := initialisationServer{dataDir, t, make([]Party, 0), false, nil}
	contents, err := ioutil.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("Error when reading data directory - %v", err)
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
			decoder := gob.NewDecoder(file)
			party := &Party{}
			err = decoder.Decode(party)
			if err != nil {
				log.Printf("Error decoding party - %v", err)
				continue
			}
			s.parties = append(s.parties, *party)
			i++
		}
	}
	return &s, nil
}

// Returns true if initialisation is complete
func (s *initialisationServer) HandleGet(w http.ResponseWriter, r *http.Request) {
	partyInitialisationData := make([]PartyInitialisationData, len(s.parties))
	for i, p := range s.parties {
		partyInitialisationData[i] = PartyInitialisationData{
			p.Name,
			fmt.Sprintf("/%d", i)}
	}
	err := s.template.Execute(w, partyInitialisationData)
	if err != nil {
		log.Fatalf("Error rendering initialisation template - %v", err)
	}
}

func (s *initialisationServer) initialiseWithNewParty(name string) error {
	s.Party = NewParty(s.dataDir, name)
	err := s.Party.Save()
	if err != nil {
		log.Printf("Error encoding party - %v", err)
	}
	log.Printf("Created new party '%s' in file '%s'", name, s.Party.Filename)
	return nil
}

func (s *initialisationServer) HandlePost(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	_, argument := getURLFunctionAndArgument(r.URL)
	if argument == "" {
		err := s.initialiseWithNewParty(r.Form["partyName"][0])
		if err != nil {
			log.Printf("Error creating file for party - %v", err)
			http.Redirect(w, r, "/", 303)
			return
		}
	} else {
		i, err := strconv.Atoi(argument)
		if err != nil {
			log.Printf("Error parsing party index - %v", err)
			http.Redirect(w, r, "/", 303)
			return
		}
		if i < 0 || i >= len(s.parties) {
			log.Printf("Party index %d out of range", i)
			http.Redirect(w, r, "/", 303)
			return
		}
		s.Party = &s.parties[i]
		log.Printf("Using existing party '%s'", s.Party.Name)
	}
	s.InitialisationComplete = true
	http.Redirect(w, r, "/roll", 303)
}
