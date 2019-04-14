package main

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetURLFunction(t *testing.T) {
	arg, _ := getURLArgument(&url.URL{Path: "/foo/"})
	assert.Equal(t, "", arg)
	arg, _ = getURLArgument(&url.URL{Path: "/bar/foo"})
	assert.Equal(t, "foo", arg)
}
