#concourse-tfe-resource
Concourse resource for Terraform Cloud and Terraform Enterprise using [Hashicorp's go-tfe library](https://github.com/hashicorp/go-tfe).

##Usage
```yaml
resource_types:
  - name: tfe
    type: docker-image
    source:
      repository: rorsten/concourse-tfe-resource
```

##Source Configuration
Name | Required | Description |
---|---|---|
organization|Yes|The name of your Terraform organization
workspace|Yes|The name of your workspace
token|Yes|An API token with at least read permission. With read permission, only in will work. With queue permissions, the `confirm` param will have no effect. Apply permission will allow full functionality. 
address|No|The URL of your Terraform Enterprise instance. Defaults to https://app.terraform.io.

##Behaviour
###`in` - Retrieve a run and related information

* Get will wait for the run to enter a final state (`policy_soft_failed`,
`planned_and_finished`, `applied`, `discarded`, `errored`, `canceled`, `force_canceled`)
* Workspace variables, environment variables and state outputs will be retrieved:
    * **IMPORTANT** - the values returned will be the current ones, even if the provided run ID is not the latest.
    * `./vars` will hold a file for each workspace variable, containing the *current* value of the variable. 
Sensitive variables will be empty.
    * `./env_vars` will hold a file for each environment variable, containing the *current* value. Sensitive values will be empty.
    * `./outputs` will hold a file for each output, containing its value.

###`out` - Push variables and create run

* Any provided variables will be pushed to the workspace, and a run will be queued.
* If the workspace is configured to auto-apply, put will return immediately. If the provided API token does not have apply permission,
the subsequent get will not complete until the run is confirmed manually.
* If the workspace requires manual confirmation and `confirm` is false, out will return and the subsequent get will
wait for manual confirmation
* If the workspace requires manual confirmation and `confirm` is true, out will wait until the run enters
a waiting state (`planned`, `cost_estimated`, `policy_checked`), out will apply the run and return. 

####Parameters
Name|Required|Description
---|---|---
vars|No|A map of workspace variables to push. See below.
env_vars|No|A map of environment variables to push. See below.
confirm|No|If true and the workspace requires confirmation, the run will be confirmed. Defaults to `false`

####Example

```yaml
    - ...
    - put: my-workspace
      params:
        vars:
          my_workspace_var:
            value: a value
            description: a description # optional 
            sensitive: true # optional, default is false
            hcl: true # optional, default is false
        env_vars:
          MY_ENV_VAR:
            value: a value
            description: a description # optional 
            sensitive: true # optional, default is false
        confirm: true # optional, default is false, see above
```
