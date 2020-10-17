package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestStartup(t *testing.T) {
	input := inputJSON{
		Source:  sourceJSON{},
		Params:  paramsJSON{},
		Version: version{},
	}

	err := startup(input)
	if err == nil || !strings.Contains(err.Error(), "creating tfe client") {
		t.Errorf("no/bad error creating client with empty config: %s", err)
	}

	input.Source.Token = os.Getenv("ATLAS_TOKEN")
	err = startup(input)

	if err == nil || !strings.Contains(err.Error(), "getting workspace") {
		t.Errorf("no/bad error without org/workspace set: %s", err)
	}

	input.Source.Workspace = "tfe-resource-test"
	input.Source.Organization = "orstensemantics"

	err = startup(input)
	if err != nil {
		t.Errorf("startup failed with valid config: %s", err)
	}
}

func TestRealMain(t *testing.T) {
	input := inputJSON{
		Source: sourceJSON{
			Workspace:    os.Getenv("TFE_WORKSPACE"),
			Organization: os.Getenv("TFE_ORGANIZATION"),
			Token:        os.Getenv("ATLAS_TOKEN"),
			Address:      os.Getenv("TFE_ADDRESS"),
		},
		Params: paramsJSON{
			PollingPeriod: 5,
		},
	}

	args := []string{"check"}
	byteInput, _ := json.Marshal(input)
	output, err := realMain(args, bytes.NewReader(byteInput))

	if err != nil {
		t.Errorf("check failed: %s", err)
	}

	wd, _ := os.Getwd()
	_ = os.Mkdir(wd+string(os.PathSeparator)+"testMainIn", os.FileMode(0755))
	run := make([]version, 1)
	_ = json.Unmarshal(output, &run)
	args = []string{"in", "testMainIn"}
	input.Version = version{Ref: run[0].Ref}
	byteInput, _ = json.Marshal(input)
	_, err = realMain(args, bytes.NewReader(byteInput))

	if err != nil {
		t.Errorf("in on checked run failed: %s", err)
	}

	args[0] = "out"
	_, err = realMain(args, bytes.NewReader(byteInput))

	if err != nil {
		t.Errorf("out failed: %s", err)
	}
}
