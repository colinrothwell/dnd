package main

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetURLFunctionAndArgument(t *testing.T) {
	f, arg := getURLFunctionAndArgument(&url.URL{Path: "/foo/"})
	assert.Equal(t, "/foo", f)
	assert.Equal(t, "", arg)
	f, arg = getURLFunctionAndArgument(&url.URL{Path: "/bar/foo"})
	assert.Equal(t, "/bar", f)
	assert.Equal(t, "foo", arg)
}
