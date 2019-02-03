package main

import (
	"dnd/creature"
	"encoding/gob"
	"io/ioutil"
	"os"
	"testing"
)

func TestGob(t *testing.T) {
	f, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatalf("Couldn't create temp file - %v", err)
	}
	fn := f.Name()
	defer os.Remove(fn)

	f, err = os.Create(fn)
	e := gob.NewEncoder(f)
	err = e.Encode(Party{"colin", "is", make([]creature.Creature, 0)})
	if err != nil {
		t.Fatalf("Error encoding party - %v", err)
	}
	f.Close()

	fi, err := os.Stat(fn)
	t.Logf("File size: %v", fi.Size())

	f, err = os.Open(fn)
	if err != nil {
		t.Fatalf("Error opening file for reading - %v", err)
	}
	d := gob.NewDecoder(f)
	var p Party
	p.Filename = "bugger"
	err = d.Decode(&p)
	if err != nil {
		t.Fatalf("Error decoding party - %v", err)
	}
	f.Close()

	if p.Filename != "colin" {
		t.Errorf("Saved party was instead '%v'", p.Filename)
	}
}

func TestPartyLoadSaving(t *testing.T) {
	d, err := ioutil.TempDir("", "encounters")
	if err != nil {
		t.Fatalf("Couldn't create temporary directory - %v", err)
	}
	is, err := newInitialisationServer(d, nil)
	if err != nil {
		t.Error(err)
	}
	is.initialiseWithNewParty("foo")
	is, err = newInitialisationServer(d, nil)
	if err != nil {
		t.Error(err)
	}
	if len(is.parties) != 1 || is.parties[0].Name != "foo" {
		t.Error("Error with stored party")
	}
	defer func() {
		err := os.RemoveAll(d)
		if err != nil {
			t.Fatalf("Error while removing temporary directory, absurdly - %v", err)
		}
	}()
}
