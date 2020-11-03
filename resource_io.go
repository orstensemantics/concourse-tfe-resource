package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/drone/envsubst"
	"github.com/hashicorp/go-tfe"
	"io"
	"log"
	"net/url"
	"os"
)

type (
	version struct {
		Ref string `json:"ref"`
	}
	sourceJSON struct {
		Workspace    string `json:"workspace"`
		Organization string `json:"organization"`
		Token        string `json:"token"`
		Address      string `json:"address"`
	}
	inputJSON struct {
		Params  paramsJSON `json:"params"`
		Source  sourceJSON `json:"source"`
		Version version    `json:"version"`
	}
	versionMetadata struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	checkOutputJSON []version
	inOutputJSON    struct {
		Version  version           `json:"version"`
		Metadata []versionMetadata `json:"metadata"`
	}
	outOutputJSON inOutputJSON
	paramsJSON    struct {
		Vars          map[string]variableJSON `json:"vars"`
		Message       string                  `json:"message"`
		Confirm       bool                    `json:"confirm"`
		PollingPeriod int                     `json:"polling_period"`
		Sensitive     bool                    `json:"sensitive"`
		ApplyMessage  string                  `json:"apply_message"`
	}
	variableJSON struct {
		File        string           `json:"file"`
		Value       string           `json:"value"`
		Description string           `json:"description"`
		Category    tfe.CategoryType `json:"category"`
		Sensitive   bool             `json:"sensitive"`
		Hcl         bool             `json:"hcl"`
	}
)

func (v variableJSON) UnmarshalJSON(b []byte) error {
	type VJ variableJSON
	var vj = (*VJ)(&v)
	vj.Category = tfe.CategoryTerraform
	if err := json.Unmarshal(b, vj); err != nil {
		return err
	}
	// for some reason this structure includes "policy-set" which you can't set as a variable
	if v.Category == tfe.CategoryTerraform || v.Category == tfe.CategoryEnv {
		return nil
	}
	return errors.New("invalid variable type")
}

func getInputs(in io.Reader) (inputJSON, error) {
	input := inputJSON{}
	input.Source = sourceJSON{
		Address: "https://app.terraform.io",
	}
	input.Params = paramsJSON{
		Message:       "Queued by ${pipeline}/${job} (${number})",
		PollingPeriod: 5,
		Sensitive:     false,
	}

	decoder := json.NewDecoder(in)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		return input, formatError(err, "parsing input")
	}

	// a few sanity checks
	misconfiguration := false

	message, err := parseMessage(input.Params.ApplyMessage)
	input.Params.ApplyMessage = message
	if err != nil {
		log.Printf("error in source configuration: invalid apply message (%s)", err)
		misconfiguration = true
	}
	message, err = parseMessage(input.Params.Message)
	input.Params.Message = message
	if err != nil {
		log.Printf("error in source configuration: invalid run message (%s)", err)
		misconfiguration = true
	}
	if _, err := url.ParseRequestURI(input.Source.Address); err != nil {
		log.Printf("error in source configuration: \"%v\" is not a valid URL", input.Source.Address)
		misconfiguration = true
	}
	if input.Source.Workspace == "" {
		log.Print("error in source configuration: workspace is not set")
		misconfiguration = true
	}
	if input.Source.Organization == "" {
		log.Print("error in source configuration: organization is not set")
		misconfiguration = true
	}
	if input.Source.Token == "" {
		log.Print("error in source configuration: token is not set")
		misconfiguration = true
	}
	if input.Params.PollingPeriod < 1 {
		log.Print("error in parameter value: polling_period must be at least 1 second")
		misconfiguration = true
	}
	if misconfiguration {
		return input, fmt.Errorf("invalid configuration provided")
	}
	return input, nil
}

func parseMessage(message string) (string, error) {
	// providing access to
	return envsubst.Eval(message, func(varName string) string {
		envVar := "NONEXISTENT_VALUE"
		switch varName {
		case "id":
			envVar = "BUILD_ID"
		case "number":
			envVar = "BUILD_NAME"
		case "job":
			envVar = "BUILD_JOB_NAME"
		case "pipeline":
			envVar = "BUILD_PIPELINE_NAME"
		case "team":
			envVar = "BUILD_TEAM_NAME"
		case "url":
			envVar = "ATC_EXTERNAL_URL"
		}
		return os.Getenv(envVar)
	})
}

func formatError(err error, context string) error {
	return fmt.Errorf("error %s: %w", context, err)
}
