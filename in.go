package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-tfe"
	"log"
	"os"
	"time"
)

func in(input inputJSON) (string, error) {
	var run *tfe.Run
	var err error
	for {
		run, err = client.Runs.Read(context.Background(), input.Version.Ref)
		if err != nil {
			return "", formatError(err, "retrieving run")
		}
		if finished(run) {
			break
		} else {
			log.Printf("Run still in progress (status = %s)", run.Status)
			time.Sleep(time.Duration(input.Params.PollingPeriod) * time.Second)
		}
	}

	output := inOutputJSON{Version: version{Ref: input.Version.Ref}}
	output.Metadata = runMetadata(input, run)

	metadataMap := make(map[string]string)
	for _, v := range output.Metadata {
		metadataMap[v.Name] = v.Value
	}
	metadataJSON, err := json.Marshal(metadataMap)
	if err != nil {
		return "", formatError(err, "marshaling metadata json")
	}

	if err = writeAndClose(fmt.Sprintf("%s%smetadata.json", workingDirectory, string(os.PathSeparator)), metadataJSON);
		err != nil {
		return "", formatError(err, "writing metadata json")
	}
	if err = writeVariables(); err != nil {
		return "", err
	}

	err = writeOutputs(input.Params.Sensitive)

	out, _ := json.Marshal(output)
	return string(out), err
}

func writeOutputs(sensitive bool) error {
	outputDir := fmt.Sprintf("%s%soutputs", workingDirectory, string(os.PathSeparator))
	if err := os.MkdirAll(outputDir, os.FileMode(0777)); err != nil {
		return formatError(err, "creating run output directory")
	}

	sv, err := client.StateVersions.Current(context.Background(), workspace.ID)
	if err != nil {
		return formatError(err, "retrieving workspace state")
	}
	stateFile, err := client.StateVersions.Download(context.Background(), sv.DownloadURL)
	if err != nil {
		return formatError(err, "downloading state file")
	}
	outputs, err := getRootOutputs(stateFile)
	if err != nil {
		return err
	}
	jsonOutput := make(map[string]json.RawMessage)
	for key, output := range outputs {
		fileName := fmt.Sprintf("%s%s%s", outputDir, string(os.PathSeparator), key)
		var outputValue json.RawMessage
		if !output.Sensitive || sensitive {
			outputValue = output.ValueRaw
		}
		jsonOutput[key] = outputValue
		if err := writeAndClose(fileName, outputValue); err != nil {
			return err
		}
	}
	jsonOutFile, err := json.Marshal(jsonOutput)
	if err != nil {
		return formatError(err, "marshaling output json")
	}
	return writeAndClose(fmt.Sprintf("%s%soutputs.json", workingDirectory, string(os.PathSeparator)), jsonOutFile)
}

func writeVariables() error {
	var (
		vars       tfe.VariableList
		err        error
		varsDir    = fmt.Sprintf("%s%s%s", workingDirectory, string(os.PathSeparator), "vars")
		envVarsDir = fmt.Sprintf("%s%s%s", workingDirectory, string(os.PathSeparator), "env_vars")
		hclVarsDir = fmt.Sprintf("%s%s%s", varsDir, string(os.PathSeparator), "hcl")
	)
	vars, err = getVariableList(client, workspace)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(hclVarsDir, os.FileMode(0777)); err != nil {
		return formatError(err, "creating output directories")
	}
	if err := os.MkdirAll(envVarsDir, os.FileMode(0777)); err != nil {
		return formatError(err, "creating output directories")
	}

	for _, v := range vars.Items {
		var fileName string
		if v.Category == tfe.CategoryEnv {
			fileName = fmt.Sprintf("%s%s%s", envVarsDir, string(os.PathSeparator), v.Key)
		} else if v.HCL {
			fileName = fmt.Sprintf("%s%s%s", hclVarsDir, string(os.PathSeparator), v.Key)
		} else {
			fileName = fmt.Sprintf("%s%s%s", varsDir, string(os.PathSeparator), v.Key)
		}
		writeAndClose(fileName, []byte(v.Value))
	}
	return nil
}

func writeAndClose(fileName string, value []byte) error {
	f, err := os.Create(fileName)
	if err != nil {
		return formatError(err, fmt.Sprintf("creating %s", fileName))
	}
	if _, err := f.Write(value); err != nil {
		return formatError(err, fmt.Sprintf("writing %s", fileName))
	}
	if err := f.Close(); err != nil {
		return formatError(err, fmt.Sprintf("closing %s", fileName))
	}
	return nil
}
