package main

import (
	"io/ioutil"
	"os"
	"testing"
)

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
	if len(is.parties) != 1 || is.parties[0].Name() != "foo" {
		t.Error("Error with stored party")
	}
	defer func() {
		err := os.RemoveAll(d)
		if err != nil {
			t.Fatalf("Error while removing temporary directory, absurdly - %v", err)
		}
	}()
}
