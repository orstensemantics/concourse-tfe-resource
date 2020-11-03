package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"strings"
	"testing"
)

func TestGetInput(t *testing.T) {
	input := inputJSON{
		Params: paramsJSON{
			PollingPeriod: -1,
			Message:       "Hiya ${fdkj",
			ApplyMessage:  "${missingbrace",
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
		if !bytes.Contains(logOutput.Bytes(), []byte("invalid apply message")) {
			t.Error("didn't complain about bad apply message")
		}
		if !bytes.Contains(logOutput.Bytes(), []byte("invalid run message")) {
			t.Error("didn't complain about bad run message")
		}
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
	input.Params.ApplyMessage = "Applying!"
	input.Params.Message = "Queued by a thing!"
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

func TestVariableJSON_UnmarshalJSON(t *testing.T) {
	var v variableJSON
	if err := v.UnmarshalJSON([]byte(`{""`)); didntErrorWithSubstr(err, "unexpected end") {
		t.Errorf("expected invalid JSON error, got %s", err)
	}
	if err := v.UnmarshalJSON([]byte(`{"category":"policy-set"}`)); didntErrorWithSubstr(err, "invalid variable type") {
		t.Errorf("expected invalid JSON error, got %s", err)
	}
	if err := v.UnmarshalJSON([]byte(`{}`)); err != nil {
		t.Errorf("expected no error, got %s", err)
	}
}

func TestParseMessage(t *testing.T) {
	message := "A message!"
	if output, err := parseMessage(message); output != message || err != nil {
		t.Errorf("message without substitutions changed: %s / %s", output, err)
	}
	os.Setenv("BUILD_ID", "Id")
	os.Setenv("BUILD_NAME", "Number")
	os.Setenv("BUILD_JOB_NAME", "Job")
	os.Setenv("BUILD_PIPELINE_NAME", "Pipeline")
	os.Setenv("BUILD_TEAM_NAME", "Team")
	os.Setenv("ATC_EXTERNAL_URL", "Url")
	message = "${id}${number}${job}${pipeline}${team}${url}"
	if output, err := parseMessage(message); output != "IdNumberJobPipelineTeamUrl" {
		t.Errorf("didn't subsitute fields as expected: %s / %s", output, err)
	}

	if output, err := parseMessage("${invalid}"); output != "" {
		t.Errorf("unexpected output with invalid field: %s / %s", output, err)
	}
}
