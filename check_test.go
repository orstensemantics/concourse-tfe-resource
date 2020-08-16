package main

import (
	"concourse-tfe-resource/common"
	mock_go_tfe "concourse-tfe-resource/mock_go_tfe"
	"encoding/json"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-tfe"
	"strconv"
	"testing"
)

func setup(t *testing.T) (
	ctrl *gomock.Controller,
	client tfe.Client,
	mockruns *mock_go_tfe.MockRuns,
	result common.CheckOutputJSON) {
	ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	client = tfe.Client{}
	mockruns = mock_go_tfe.NewMockRuns(ctrl)
	client.Runs = mockruns

	result = common.CheckOutputJSON{}
	return
}

func runList(start int, len int) (list tfe.RunList) {
	for i := 0; i < len; i += 1 {
		list.Items = append(list.Items, &tfe.Run{ID: strconv.Itoa(i + start)})
	}
	return
}

func TestCheckWithNilVersion(t *testing.T) {
	_, client, mockruns, result := setup(t)

	no_version_call := runList(0, 5)

	mockruns.EXPECT().List(gomock.Any(), gomock.Eq("foo"), gomock.Any()).Return(&no_version_call, nil)

	output := check(common.InputJSON{}, &client, "foo")

	json.Unmarshal([]byte(output), &result)

	if len(result) != 1 {
		t.Errorf("check with nil version returned %d elements", len(result))
	} else if result[0].Ref != "0" {
		t.Errorf("check with nil version didn't return the first result")
	}
}

func TestCheckWithExistingVersion(t *testing.T) {
	_, client, mockruns, result := setup(t)

	no_version_call := runList(0, 5)

	mockruns.EXPECT().List(gomock.Any(), gomock.Eq("foo"), gomock.Any()).Return(&no_version_call, nil)
	output := check(common.InputJSON{Version: common.Version{Ref: "2"}}, &client, "foo")

	json.Unmarshal([]byte(output), &result)

	if len(result) != 3 {
		t.Errorf("check with third oldest version returned %d elements", len(result))
	} else if result[2].Ref != "2" {
		t.Errorf("check with third oldest version didn't return the expected elements")
	}
}

func TestCheckWithVersionOnSecondPage(t *testing.T) {
	_, client, mockruns, result := setup(t)

	no_version_call := runList(0, 5)
	second_call := runList(5, 5)

	rlo1 := tfe.RunListOptions{ListOptions: tfe.ListOptions{PageSize: 100, PageNumber: 0}}
	rlo2 := tfe.RunListOptions{ListOptions: tfe.ListOptions{PageSize: 100, PageNumber: 1}}
	mockruns.EXPECT().List(gomock.Any(), gomock.Eq("foo"), gomock.Eq(rlo1)).Return(&no_version_call, nil)
	mockruns.EXPECT().List(gomock.Any(), gomock.Eq("foo"), gomock.Eq(rlo2)).Return(&second_call, nil)
	output := check(common.InputJSON{Version: common.Version{Ref: "8"}}, &client, "foo")

	json.Unmarshal([]byte(output), &result)
	if len(result) != 9 {
		t.Errorf("check for 9th most recent version (multiple calls) returned %d results", len(result))
	} else if result[8].Ref != "8" {
		t.Errorf("multiple call check returns incorrect results")
	}
}

func TestCheckWithNonexistentVersion(t *testing.T) {
	_, client, mockruns, result := setup(t)

	no_version_call := runList(0, 5)

	// if the provided version does not seem to exist, return the current version
	rlo1 := tfe.RunListOptions{ListOptions: tfe.ListOptions{PageSize: 100, PageNumber: 0}}
	rlo2 := tfe.RunListOptions{ListOptions: tfe.ListOptions{PageSize: 100, PageNumber: 1}}
	mockruns.EXPECT().List(gomock.Any(), gomock.Eq("foo"), gomock.Eq(rlo1)).Return(&no_version_call, nil)
	mockruns.EXPECT().List(gomock.Any(), gomock.Eq("foo"), gomock.Eq(rlo2)).Return(&tfe.RunList{}, nil)
	output := check(common.InputJSON{Version: common.Version{Ref: "8"}}, &client, "foo")

	json.Unmarshal([]byte(output), &result)

	if len(result) != 1 {
		t.Errorf("check with non-present version returned %d elements", len(result))
	} else if result[0].Ref != "0" {
		t.Errorf("check with non-present version didn't return the first result")
	}
}
