package common

import (
	"encoding/json"
	"log"
	"os"
)

type (
	Version struct {
		Ref string `json:"ref"`
	}
	// InputJSON ...
	InputJSON struct {
		Params  map[string]string `json:"params"`
		Source  map[string]string `json:"source"`
		Version Version           `json:"Version"`
	}
	Metadata struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	CheckOutputJSON []Version
	InOutputJSON    struct {
		Version  Version    `json:"Version"`
		Metadata []Metadata `json:"metadata"`
	}
	OutOutputJSON InOutputJSON
)

func GetInputs() (input InputJSON) {
	decoder := json.NewDecoder(os.Stdin)
	err := decoder.Decode(&input)
	if err != nil {
		log.Fatalf("Failed to parse input: %s", err)
	}

	return
}

func ValidateSource(input InputJSON) bool {
	var aok bool
	mandatory := [...]string{"workspace", "organization", "token"}

	for _, v := range mandatory {
		if _, ok := input.Source[v]; !ok {
			log.Printf("ERROR: Missing required source parameter \"%s\"", v)
			aok = false
		}
	}
	return aok
}

