# concourse-tfe-resource
Concourse resource for Terraform Cloud and Terraform Enterprise using [Hashicorp's go-tfe library](https://github.com/hashicorp/go-tfe).

## Usage
```yaml
resource_types:
  - name: tfe
    type: docker-image
    source:
      repository: rorsten/concourse-tfe-resource
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
* Get will *not* fail based on the final state of the run. 
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
Name|Required|Description|Default
---|---|---|---|
polling_period|No|How many seconds to wait between API calls while waiting for runs to reach final states when getting a run.|5
sensitive|No|Whether to include values for sensitive outputs.

### `out` - Push variables and create run

* Any provided variables will be pushed to the workspace, and a run will be queued.
* If the workspace is configured to auto-apply, put will return immediately. If the provided API token does not have apply permission,
the subsequent get will not complete until the run is confirmed manually.
* If the workspace requires manual confirmation and `confirm` is false, out will return and the subsequent get will
wait for manual confirmation
* If the workspace requires manual confirmation and `confirm` is true, out will wait until the run enters
a waiting state (`planned`, `cost_estimated`, `policy_checked`), out will apply the run and return. 

#### Parameters
Name|Required|Description
---|---|---
vars|No|A map of workspace variables to push. See below.
env_vars|No|A map of environment variables to push. See below.
confirm|No|If true and the workspace requires confirmation, the run will be confirmed. Defaults to `false`

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
        env_vars:
          MY_ENV_VAR:
            value: a value
            file: someoutput/filename
            description: a description # optional 
            sensitive: true # optional, default is false
        confirm: true # optional, default is false, see above
        message: Name of Build in Terraform Cloud # optional
        # default message is "Queued by {pipeline_name}/{job_name} ({build_number})" 
```
