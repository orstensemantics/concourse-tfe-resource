# concourse-tfe-resource

[![Go Report Card](https://goreportcard.com/badge/github.com/orstensemantics/concourse-tfe-resource)](https://goreportcard.com/report/github.com/orstensemantics/concourse-tfe-resource)
[![Maintainability](https://api.codeclimate.com/v1/badges/7dd55f613030fef89324/maintainability)](https://codeclimate.com/github/orstensemantics/concourse-tfe-resource/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/7dd55f613030fef89324/test_coverage)](https://codeclimate.com/github/orstensemantics/concourse-tfe-resource/test_coverage)
[![Dependencies](https://img.shields.io/librariesio/github/orstensemantics/concourse-tfe-resource)](https://libraries.io/github/orstensemantics/concourse-tfe-resource)
![Build status](https://github.com/orstensemantics/concourse-tfe-resource/workflows/tests/badge.svg)
![Docker build status](https://img.shields.io/docker/cloud/build/orstensemantics/concourse-tfe-resource)
![Docker Image Size](https://img.shields.io/docker/image-size/orstensemantics/concourse-tfe-resource)

Concourse resource for Terraform Cloud and Terraform Enterprise using [Hashicorp's go-tfe library](https://github.com/hashicorp/go-tfe).

## Usage
```yaml
resource_types:
  - name: tfe
    type: docker-image
    source:
      repository: orstensemantics/concourse-tfe-resource
```

## Source Configuration
Name | Required | Description |
---|---|---|
organization|Yes|The name of your Terraform organization
workspace|Yes|The name of your workspace
token|Yes|An API token with at least read permission. With read permission, only in and check will work. With queue permissions, the `confirm` param will have no effect. Apply permission will allow full functionality. 
address|No|The URL of your Terraform Enterprise instance. Defaults to https://app.terraform.io.

## Behaviour
### `in` - Retrieve a run and related information

* Get will wait for the run to enter a final state (`policy_soft_failed`,
`planned_and_finished`, `applied`, `discarded`, `errored`, `canceled`, `force_canceled`)
* Get will *not* fail based on the final state of the run. If you need to respond to the final state, this will exit with
a non-zero status if the run didn't end with a successful apply or a plan with no changes:
 ```shell script
 $ cat your_run/metadata.json | jq -e '.final_status | IN(["applied","planned_and_finished"], .)'
 ```
* If the run requires confirmation to apply and `confirm` is `true`, get will apply the run.
    * This is determined by the `actions.is-confirmable` attribute of the run and *not* the auto-apply setting of the
    workspace, so this will apply to runs created by workspace triggers
* Workspace variables, environment variables and state outputs will be retrieved:
    * **IMPORTANT** - the values returned will be the current ones, even if the provided run ID is not the latest.
    * `./vars` will hold a file for each workspace variable, containing the *current* value of the variable. HCL
     variables will be in `.vars/hcl`. Sensitive variables will be empty.
    * `./env_vars` will hold a file for each environment variable, containing the *current* value. Sensitive values
     will be empty.
    * `./outputs.json` will be a JSON file of all of the root level outputs of the *current* workspace state. Sensitive
     values will be empty strings unless the `sensitive` param is true. Suitable for load_var/set_pipeline steps.
    * `./outputs` will hold a file for each root level output of the *current* workspace state. Sensitive values will be
    empty files unless the `sensitive` param is true. Since outputs can be complex values, the contents of the file are
    JSON, so simple string outputs are quoted.
    * `./metadata.json` will contain the same metadata values visible in the resource version.

#### Parameters
Name|Description|Default
---|---|---|
polling_period|How many seconds to wait between API calls while waiting for runs to reach final states when getting a run.|5
sensitive|Whether to include values for sensitive outputs.|`false`
confirm|If true and the workspace requires confirmation, the run will be confirmed.|`false`
apply_message|Comment to include while confirming the run. See below for available variables.|

### `out` - Push variables and create run

* Any provided variables will be pushed to the workspace
* A run will be queued.

#### Parameters
Name|Description
---|---
vars|A map of workspace variables to push.
message|Message to describe the run. Defaults to "Queued by ${pipeline}/${job} (${number})". See below for available variables.

#### Variable Parameters

At least one of `value` or `file` must be set for every entry. All others are optional.

Name|Default|Description
---|---|---
value| |A string value for the variable. Takes precedence over `file`.
file| |Relative path to a file containing a value to set. Ignored if `value` is set.
description| |A description of the variable.
category|`terraform`|Change to `env` to push an environment variable instead of a terraform variable. Only `terraform` and `env` are valid.
sensitive|`false`|If `true`, the variable value will be hidden
hcl|`false`|If `true`, the variable will be treated as


#### Example

```yaml
    - ...
    - put: my-workspace
      params:
        vars:
          my_workspace_var:
            # if you specify value *and* file, value will take precedence
            value: a value # you can specify a value directly
            file: someoutput/filename # or you can reference a file containing the value
            description: a description # optional 
            sensitive: true # optional, default is false
            hcl: true # optional, default is false
          MY_ENV_VAR:
            value: a value
            file: someoutput/filename
            category: env
        message: Name of Build in Terraform Cloud # optional
```

###Message Variables
The `message` and `apply_message` variables support interpolations via [drone/envsubst](https://github.com/drone/envsubst).
The table below lists the available variables. Most bash string replacement functions are supported (see the link for more details).

Variable|Description|Concourse Environment Variable
---|---|---
url|The base URL of the concourse server.|ATC_EXTERNAL_URL
team|The name of the team owning the pipeline. |BUILD_TEAM_NAME
pipeline|The name of the pipeline.|BUILD_PIPELINE_NAME
job|The name of the job.|BUILD_JOB_NAME
number|The number of the build (e.g., 14.2)|BUILD_NAME
id|The concourse internal build ID.|BUILD_ID
