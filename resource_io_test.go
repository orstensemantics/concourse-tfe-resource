package main

import (
	"bytes"
	"encoding/json"
	"log"
	"strings"
	"testing"
)

func TestGetInput(t *testing.T) {
	input := inputJSON{
		Params: paramsJSON{
			PollingPeriod: -1,
		},
		Source: sourceJSON{
			Workspace:    "",
			Organization: "",
			Token:        "",
			Address:      "",
		},
		Version: version{
			Ref: "",
		},
	}
	var logOutput bytes.Buffer
	log.SetOutput(&logOutput)

	inputBytes, _ := json.Marshal(input)
	_, err := getInputs(bytes.NewReader(inputBytes))

	if err == nil {
		t.Error("accepted bad input")
	} else {
		if !bytes.Contains(logOutput.Bytes(), []byte("is not a valid URL")) {
			t.Error("didn't complain about bad url")
		}
		if !bytes.Contains(logOutput.Bytes(), []byte("workspace is not set")) {
			t.Error("didn't complain about empty workspace")
		}
		if !bytes.Contains(logOutput.Bytes(), []byte("organization is not set")) {
			t.Error("didn't complain about empty organization")
		}
		if !bytes.Contains(logOutput.Bytes(), []byte("token is not set")) {
			t.Error("didn't complain about empty token")
		}
		if !bytes.Contains(logOutput.Bytes(), []byte("must be at least 1 second")) {
			t.Error("didn't complain about bad polling_period")
		}
	}

	input.Source.Address = "https://foo.bar"
	input.Source.Workspace = "workspace"
	input.Source.Organization = "org"
	input.Source.Token = "token"
	input.Params.PollingPeriod = 4
	logOutput.Reset()
	inputBytes, _ = json.Marshal(input)
	_, err = getInputs(bytes.NewReader(inputBytes))
	if err != nil {
		t.Error("returned error with valid config")
	}

	inputBytes = []byte(`{"params":{"bnoggle":"farf"},"version":{"ref":"foo"}}`)
	_, err = getInputs(bytes.NewReader(inputBytes))
	if err == nil || !strings.Contains(err.Error(), "bnoggle") {
		t.Error("didn't complain about invalid field")
	}
}
