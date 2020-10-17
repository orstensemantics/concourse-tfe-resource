package main

import (
	"encoding/json"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-tfe"
	"strconv"
	"testing"
)

func runList(start int, len int) (list tfe.RunList) {
	for i := 0; i < len; i++ {
		list.Items = append(list.Items, &tfe.Run{ID: strconv.Itoa(i + start)})
	}
	return
}

func TestCheckWithNilVersion(t *testing.T) {
	setup(t)
	result := checkOutputJSON{}

	firstCall := runList(0, 5)
	input := inputJSON{Source: sourceJSON{Workspace: "foo"}}

	runs.EXPECT().List(gomock.Any(), gomock.Eq(workspace.ID), gomock.Any()).Return(&firstCall, nil)
	runs.EXPECT().List(gomock.Any(), gomock.Eq(workspace.ID), gomock.Any()).Return(&tfe.RunList{}, nil)

	output, _ := check(input)

	json.Unmarshal([]byte(output), &result)

	if len(result) != 5 {
		t.Errorf("check with nil version returned %d elements", len(result))
	} else if result[0].Ref != "4" {
		t.Errorf("check with nil version didn't return the first result")
	}
}

func TestCheckWithExistingVersion(t *testing.T) {
	setup(t)
	result := checkOutputJSON{}

	firstCall := runList(0, 5)
	input := inputJSON{Source: sourceJSON{Workspace: "foo"}}

	runs.EXPECT().List(gomock.Any(), gomock.Eq("foo"), gomock.Any()).Return(&firstCall, nil)
	input.Version.Ref = "2"
	output, _ := check(input)

	json.Unmarshal([]byte(output), &result)

	if len(result) != 3 {
		t.Errorf("check with third oldest version returned %d elements", len(result))
	} else if result[0].Ref != "2" {
		t.Errorf("check with third oldest version didn't return the expected elements")
	}
}

func TestCheckWithVersionOnSecondPage(t *testing.T) {
	setup(t)
	result := checkOutputJSON{}

	firstCall := runList(0, 5)
	secondCall := runList(5, 5)
	input := inputJSON{Source: sourceJSON{Workspace: "foo"}}

	rlo1 := tfe.RunListOptions{ListOptions: tfe.ListOptions{PageSize: 100, PageNumber: 0}}
	rlo2 := tfe.RunListOptions{ListOptions: tfe.ListOptions{PageSize: 100, PageNumber: 1}}
	runs.EXPECT().List(gomock.Any(), gomock.Eq("foo"), gomock.Eq(rlo1)).Return(&firstCall, nil)
	runs.EXPECT().List(gomock.Any(), gomock.Eq("foo"), gomock.Eq(rlo2)).Return(&secondCall, nil)
	input.Version.Ref = "8"
	output, _ := check(input)

	json.Unmarshal([]byte(output), &result)
	if len(result) != 9 {
		t.Errorf("check for 9th most recent version (multiple calls) returned %d results", len(result))
	} else if result[0].Ref != "8" {
		t.Errorf("multiple call check returns incorrect results")
	}
}

func TestCheckWithNonexistentVersion(t *testing.T) {
	setup(t)
	result := checkOutputJSON{}

	firstCall := runList(0, 5)
	input := inputJSON{Source: sourceJSON{Workspace: "foo"}}

	// if the provided version does not seem to exist, return the current version
	rlo1 := tfe.RunListOptions{ListOptions: tfe.ListOptions{PageSize: 100, PageNumber: 0}}
	rlo2 := tfe.RunListOptions{ListOptions: tfe.ListOptions{PageSize: 100, PageNumber: 1}}
	runs.EXPECT().List(gomock.Any(), gomock.Eq("foo"), gomock.Eq(rlo1)).Return(&firstCall, nil)
	runs.EXPECT().List(gomock.Any(), gomock.Eq("foo"), gomock.Eq(rlo2)).Return(&tfe.RunList{}, nil)
	input.Version.Ref = "8"
	output, _ := check(input)

	json.Unmarshal([]byte(output), &result)

	if len(result) != 1 {
		t.Errorf("check with non-present version returned %d elements", len(result))
	} else if result[len(result)-1].Ref != "0" {
		t.Errorf("check with non-present version didn't return the first result")
	}
}

func TestCheckWithFailingListCall(t *testing.T) {
	setup(t)
	result := checkOutputJSON{}

	firstCall := runList(0, 5)
	input := inputJSON{Source: sourceJSON{Workspace: "foo"}}

	rlo1 := tfe.RunListOptions{ListOptions: tfe.ListOptions{PageSize: 100, PageNumber: 0}}
	runs.EXPECT().List(gomock.Any(), gomock.Eq("foo"), gomock.Eq(rlo1)).Return(&firstCall, errors.New("NO"))
	output, err := check(input)

	if output != nil || err == nil || err.Error() != "error listing runs: NO" {
		t.Errorf("unexpected:\n\tresult = \"%s\"\n\terr = \"%s\"", result, err)
	}
}
