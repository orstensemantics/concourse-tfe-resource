package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestGetInput(t *testing.T)  {
	input := inputJSON{
		Params:  paramsJSON{
			PollingPeriod: 1,
		},
		Source:  sourceJSON{
			Workspace:    "a",
			Organization: "b",
			Token:        "c",
			Address:      "",
		},
		Version: version{
			Ref: "",
		},
	}

	inputBytes, _ := json.Marshal(input)
	_, err := getInputs(bytes.NewReader(inputBytes))

	if err == nil {
		t.Error("accepted input without address")
	} else if !strings.Contains(err.Error(), "error parsing source address") {
		t.Errorf("wrong error message: %s", err)
	}

	input.Source.Address = "https://foo.bar"
	input.Source.Workspace = ""
	inputBytes, _ = json.Marshal(input)
	_, err = getInputs(bytes.NewReader(inputBytes))
	if err == nil {
		t.Error("accepted missing workspace")
	} else if !strings.Contains(err.Error(), "fields must be set") {
		t.Error("accepted blank config field")
	}

	input.Source.Workspace = "a"
	input.Params.PollingPeriod = 0
	inputBytes, _ = json.Marshal(input)
	_, err = getInputs(bytes.NewReader(inputBytes))
	if err == nil {
		t.Error("accepted missing workspace")
	} else if !strings.Contains(err.Error(), "polling_period must be") {
		t.Error("accepted zero polling period")
	}

	input.Params.PollingPeriod = 1
	inputBytes, _ = json.Marshal(input)
	_, err = getInputs(bytes.NewReader(inputBytes))
	if err != nil {
		t.Error("errored on good config")
	}

	inputBytes = []byte(`{"params":{"bnoggle":"farf"},"version":{"ref":"foo"}}`)
	_, err = getInputs(bytes.NewReader(inputBytes))
	if err == nil || !strings.Contains(err.Error(), "bnoggle") {
		t.Error("didn't complain about invalid field")
	}
}