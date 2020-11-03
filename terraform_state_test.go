package main

import (
	"strings"
	"testing"
)

func TestGetRootOutputs(t *testing.T) {
	stateFile := ""

	outputs, err := getRootOutputs([]byte(stateFile))
	if stateDidntErrorWithSubstr(outputs, err, "decoding state file") {
		t.Error("accepted an empty state")
	}

	stateFile = `{"version":1}`
	outputs, err = getRootOutputs([]byte(stateFile))
	if stateDidntErrorWithSubstr(outputs, err, "unsupported state version") {
		t.Error("accepted unsupported state version")
	}

	stateFile = `{"version":2,"modules":"pods"}`
	outputs, err = getRootOutputs([]byte(stateFile))
	if stateDidntErrorWithSubstr(outputs, err, "decoding v2 state") {
		t.Error("accepted invalid v2 state")
	}

	stateFile = `{"version":2,"modules":[]}`
	outputs, err = getRootOutputs([]byte(stateFile))
	if stateDidntErrorWithSubstr(outputs, err, "no root module") {
		t.Error("accepted invalid v2 module list")
	}

	stateFile = `{"version":4,"outputs":"pods"}`
	outputs, err = getRootOutputs([]byte(stateFile))
	if stateDidntErrorWithSubstr(outputs, err, "decoding v4 state") {
		t.Error("accepted invalid v4 state")
	}

}

func stateDidntErrorWithSubstr(outputs map[string]outputStateV4, err error, expected string) bool {
	return outputs != nil || err == nil || !strings.Contains(err.Error(), expected)
}
