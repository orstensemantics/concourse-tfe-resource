package main

import (
	"encoding/json"
	"fmt"
)

// This file exists because go-tfe doesn't currently support the state outputs api for terraform cloud.
// It does, however, give you urls to download raw state, so I borrowed structure definitions from

type (
	stateVersion struct {
		Version int `json:"version"`
	}

	// Poaching a subset of state handling structs from terraform's code
	stateV4 struct {
		Version     int                      `json:"version"`
		RootOutputs map[string]outputStateV4 `json:"outputs"`
	}

	outputStateV4 struct {
		ValueRaw     json.RawMessage `json:"value"`
		ValueTypeRaw json.RawMessage `json:"type"`
		Sensitive    bool            `json:"sensitive,omitempty"`
	}

	stateV2 struct {
		// Version is the state file protocol version.
		Version int `json:"version"`

		// Modules contains all the modules in a breadth-first order
		Modules []*moduleStateV2 `json:"modules"`
	}

	moduleStateV2 struct {
		// Path is the import path from the root module. Modules imports are
		// always disjoint, so the path represents amodule tree
		Path []string `json:"path"`

		// Outputs declared by the module and maintained for each module
		// even though only the root module technically needs to be kept.
		// This allows operators to inspect values at the boundaries.

		// treating this as v4 because the fields are the same
		Outputs map[string]outputStateV4 `json:"outputs"`
	}
)

func getRootOutputs(stateFile []byte) (map[string]outputStateV4, error)  {
	var s stateVersion
	if err := json.Unmarshal(stateFile, &s); err != nil {
		return nil, formatError(err, "decoding state file")
	}

	if s.Version == 2 || s.Version == 3 {
		return getStateV2Outputs(stateFile)
	} else if s.Version == 4 {
		return getStateV4Outputs(stateFile)
	} else {
		return nil, fmt.Errorf("error reading state outputs: unsupported state version")
	}
}

func getStateV2Outputs(stateFile []byte) (map[string]outputStateV4, error) {
	var state stateV2
	if err := json.Unmarshal(stateFile, &state); err != nil {
		return nil, formatError(err, "decoding v2 state file")
	}

	// in v2/v3 state, Modules is a breadth-first array, so 0 is the root module
	return state.Modules[0].Outputs, nil
}

func getStateV4Outputs(stateFile []byte) (map[string]outputStateV4, error) {
	var state stateV4
	if err := json.Unmarshal(stateFile, &state); err != nil {
		return nil, formatError(err, "decoding v4 state file")
	}

	// v4 state exposes the root outputs directly
	return state.RootOutputs, nil
}
