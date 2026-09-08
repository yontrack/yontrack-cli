Yontrack CLI
============

[![Build](https://github.com/yontrack/yontrack-cli/actions/workflows/go.yml/badge.svg)](https://github.com/yontrack/yontrack-cli/actions/workflows/go.yml)

[Yontrack](https://github.com/nemerosa/ontrack) is an application which store all events which happen in your CI/CD environment: branches, builds, validations, promotions, labels, commits, etc. It allows your delivery chains to reach new levels by driving your pipelines using real-time data.

The Yontrack CLI is a Command Line Interface tool, available on many platforms, which allows you to feed information into Yontrack from any shell platform.

> Yontrack was previously called Ontrack. The GitHub organisation, the companion repositories and the GraphQL schema file still carry the old name; links and commands below use it deliberately and are not stale.

> The Ontrack CLI works only with the version 4 of [Ontrack](https://github.com/nemerosa/ontrack).

# Installation

On macOS and Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/yontrack/yontrack-cli/main/install.sh | sh
```

This works out your platform, downloads the matching binary, checks it against the checksums published with the release, and installs it as `yontrack`. Run it again to upgrade. The CLI is a single Go binary with no dependencies.

It installs into `$HOME/.local/bin`, so it never needs a password. If that directory is not on your `PATH`, the script says so and prints the line to add to your shell's startup file.

Three environment variables control it:

| Variable | Default | |
|---|---|---|
| `INSTALL_DIR` | `$HOME/.local/bin` | Where to install. The script never asks for a password: if it cannot write there, it installs nothing and prints what to run instead. |
| `VERSION` | the latest release | Install a specific release tag rather than the newest one. |
| `BASE_URL` | GitHub releases | Fetch from somewhere else, such as an internal mirror. |

To install somewhere else — a system-wide directory, say — set `INSTALL_DIR`. A system directory is not yours to write to, so that one needs `sudo`, and the script will not be piped into it: download it, read it, then run it.

```bash
curl -fsSLo install.sh https://raw.githubusercontent.com/yontrack/yontrack-cli/main/install.sh
sudo INSTALL_DIR=/usr/local/bin sh install.sh
```

## Windows

Download `yontrack-windows-amd64.exe` from the [releases](https://github.com/yontrack/yontrack-cli/releases) page and rename it to `yontrack.exe`.

## Downloading the binary yourself

Installing by hand stays supported — for air-gapped machines, an internal mirror, or a pinned CI image. Take the asset for your platform from the [releases](https://github.com/yontrack/yontrack-cli/releases) page, then do what the download does not do for you:

```bash
mv ./yontrack-darwin-arm64 ./yontrack        # the asset is named for its platform
chmod +x ./yontrack
xattr -d com.apple.quarantine ./yontrack     # macOS only
```

The last command reports `No such xattr` if the attribute is not there, which is harmless — a file fetched with `curl` never has one.

The `xattr` command needs explaining. A browser tags whatever it downloads with a `com.apple.quarantine` attribute, and macOS refuses to run a quarantined binary that Apple has not notarized — the dialog says it "could not verify" the binary "is free of malware". `curl` never sets that attribute, which is why the installer above does not meet the dialog at all. Removing it by hand leaves the file in the same position as one `curl` fetched. That is a real Gatekeeper check you are clearing, so do it only for a binary you fetched from the releases page above. The alternative, since macOS Sequoia removed the Control-click override, is System Settings → Privacy & Security, once per binary, with an admin password.

Run `yontrack` from a terminal. Double-clicking a *quarantined* binary in Finder runs a separate Gatekeeper check that fails however well signed it is, so if you skip the `xattr` step that route stays closed.

# Setup

You need to register a configuration:

```bash
yontrack config create prod https://ontrack.example.com --token <token>
```

This registers an installation called `prod`, located at https://ontrack.example.com, using an authentication token.

The configuration is stored on disk, in `~/.yontrack-config.yaml` and the `config create` needs to be done only once.

> The Ontrack CLI supports only version 4.x and beyond of Ontrack.

## Managing configurations

List all registered configurations:

```bash
yontrack config list
```

Switch to a different configuration:

```bash
yontrack config select <name>
```

Temporarily disable or re-enable a configuration without removing it:

```bash
yontrack config disable <name>
yontrack config enable <name>
```

Delete a configuration permanently:

```bash
yontrack config delete <name>
```

> If the deleted configuration was the currently selected one, you will need to run `config select` to choose another before using the CLI.

# Usage

After the configuration has been set, injection of data into Yontrack from a CI pipeline can be typically done this way.

## CI setup

> This step supersedes most of the "setup" commands.

An easy way to setup the project, branch and build is to use the `ci config` command:

```shell
yontrack ci config
```

By default, the `ci config` command will use a file at `.yontrack/ci.yaml` in the current directory
but this can be changed using the `--file` option.

The `ci config` command will create the project, branch and build if they do not exist, based on local 
information extracted from the environment. For security reasons, no environment variable is used by default
and they must be passed explicitly using the `--env` options:

```shell
yontrack ci config \
  --env GIT_URL=git@github.com:nemerosa/ontrack.git \
  --env GIT_BRANCH=release/5.0
```

The environment variables can also be be set into 
a local environment file, for example:

```text
GIT_URL=git@github.com:nemerosa/ontrack.git
GIT_BRANCH=release/5.0
```

and then passed to the `ci config` command using the `--env-file` option:

```shell
yontrack ci config \
  --env-file .env
```

where `.env` is the name of the file containing the environment variables above.

The very minimal YAML configuration is:

```yaml
version: v1
configuration: { }
```

When using this configuration, Yontrack will create the project, the branch and the build, based on the information
found in the CI context (mostly, the environment variables provided by Jenkins).

You can of course configure way more, like the promotions and validation stamps at the branch level,
with some additional configuration for the release branches:

```yaml
version: v1
configuration:
  branch:
    validations:
      unit-tests:
        tests: { }
    promotions:
      BRONZE:
        validations:
          - unit-tests
```

> For more information about the configuration, see the Yontrack documentation to see how to configure:
> properties, promotions, validation stamps, notifications, workflows, auto-versioning, custom setup, etc.

### Templating

The configuration file (defaulting to `.yontrack/ci.yaml`) is processed as a template before being parsed.

#### File inclusion

You can include other YAML files by using the `@path` syntax as a value:

```yaml
version: v1
configuration:
  branch:
    validations: '@.yontrack/validations.yaml'
```

If the path is relative, it is resolved relative to the file containing the reference.

#### Variables

You can access variables passed via the `--var` option:

```shell
yontrack ci config --var version=1.0.0
```

And then in your configuration:

```yaml
version: v1
configuration:
  build:
    name: "{{ vars \"version\" }}"
```

You can also use a default value if the variable is not set:

```yaml
name: "{{ getvar \"version\" \"development\" }}"
```

#### Environment variables

Similarly, you can access environment variables passed via `--env` or `--env-file`:

```yaml
version: v1
configuration:
  project:
    name: "{{ env \"GIT_URL\" | base }}"
```

With default values:

```yaml
version: v1
configuration:
  project:
    name: "{{ getenv \"PROJECT_NAME\" \"my-project\" }}"
```

#### Sprig functions

The [Sprig](https://masterminds.github.io/sprig/) library is integrated, providing many useful functions for string manipulation, math, and more.

```yaml
name: "{{ vars \"name\" | upper | truncate 10 }}"
```

## Configuration and using it in other commands

Most of the other commands are able to use the following
environment variables to replace explicit parameters:

* `YONTRACK_PROJECT_NAME` instead of `--project`
* `YONTRACK_BRANCH_NAME` instead of `--branch`
* `YONTRACK_BUILD_NAME` instead of `--build`

The `ci config` command will export these environment variables for you, and it can be as easy as:

```shell
eval $(yontrack ci config --output ...)
```

or:

```shell
yontrack ci config --output ... > .yontrack
source .yontrack
```

## Branch setup

We make sure the branch managed by the pipeline is registered into Yontrack:

```bash
# Setup of the branch
yontrack branch setup --project <project> --branch <branch>
```

Here, `<project>` is the name of your project or repository, and `<branch>` is typically the Git branch name
or the PR name (like `PR-123`). The `branch setup` operation is idempotent.

> Run `yontrack branch setup --help` for additional options.

## Validation stamps setup

The CLI can be used to create validation stamps:

```bash
yontrack validation-stamp setup --project <project> --branch <branch> --validation <validation>
```

The `validation-stamp setup` (or `vs setup` for a shortcut) command is idempotent.

Additionally, a validation stamp can be created with a
[data type](https://docs.yontrack.com/yontrack/ref/latest/content/concepts/model/index.html#validation-stamp-types)
and its configuration. For example, to create a CHML validation type:

```bash
yontrack validation-stamp setup --project <project> --branch <branch> --validation <validation> \
    --data-type net.nemerosa.ontrack.extension.general.validation.CHMLValidationDataType \
    --data-config '{warningLevel: {level: "HIGH",value:1},failedLevel:{level:"CRITICAL",value:1}}'
```

The later syntax is pretty cumbersome and the CLI provides dedicated commands for the most used data types:

* for CHML data type:

```bash
yontrack validation-stamp setup --project <project> --branch <branch> --validation <validation> \
    chml \
        --warning HIGH=1 \
        --failed CRITICAL=1
```

* for test summary data type:

```bash
yontrack validation-stamp setup --project <project> --branch <branch> --validation <validation> \
    tests --warning-if-skipped true
```

* for percentage data type:

```bash
yontrack validation-stamp setup --project <project> --branch <branch> --validation <validation> \
    percentage \
        --warning 60 \
        --failure 50 \
        --ok-if-greater false
```

* for metrics data type:

```bash
yontrack validation-stamp setup --project <project> --branch <branch> --validation <validation> \
    metrics
```

## Promotions and auto promotion

Promotions can be created using:

```bash
yontrack promotion-level setup --project <project> --branch <branch> --promotion <promotion>
```

Their auto promotion can be set using:

```bash
yontrack promotion-level setup --project <project> --branch <branch> --promotion <promotion> \
   --validation <stamp1> \
   --validation <stamp2> \
   --depends-on <other-promotion-1> \
   --depends-on <other-promotion-2>
```

The validation stamps and promotions this command depends on will be created if they don't exist already.

## Build setup

Then, you can create a build entry the same way:

```bash
# Setup of the build
yontrack build setup --project <project> --branch <branch> --build <build>
```

where `<build>` is a unique identifier for your build (typically a build number).

If you need to associated a release label to your build, you can use the `--release` option:

```bash
yontrack build setup --project <project> --branch <branch> --build <build> --release <label>
```

The same way, you can associate a Git commit property to the build with the `--commit` option:

```bash
yontrack build setup --project <project> --branch <branch> --build <build> --commit <commit>
```

## Searching for builds

`build search` finds builds in a project and prints them, one per line:

```bash
yontrack build search --project my-project --branch main --count 1
```

Restrict the search using any combination of criteria. To get the last build to have been promoted:

```bash
yontrack build search --project my-project --branch main --with-promotion BRONZE --count 1
```

To find a build by its display name - its release label when it has one, and its own name otherwise:

```bash
yontrack build search --project my-project --branch main --with-display-name 1.2.3
```

The display name is matched case insensitively and *partially*, so `1.2` would also match `1.2.3`. Anchor the pattern with `^` and `$` for an exact match, remembering that dots are wildcards:

```bash
yontrack build search --project my-project --branch main --with-display-name '^1\.2\.3$'
```

`--with-display-name` requires `--branch`.

To find builds carrying a given property, named by its type:

```bash
yontrack build search --project my-project --branch main \
    --with-property net.nemerosa.ontrack.extension.general.ReleasePropertyType \
    --with-property-value 1.2.3
```

Without `--with-property-value`, any build carrying the property matches, whatever its value. Searching on a Git commit has its own shorthand:

```bash
yontrack build search --project my-project --branch main --commit c1f6c19
```

By default only build names are printed. `--display-id` prints their IDs instead, and with `--count 1` the whole build can be exported:

```bash
yontrack build search --project my-project --branch main --count 1 --output json
yontrack build search --project my-project --branch main --count 1 --output env
```

Use `--accept-not-found` (or `-n`) to get an empty result rather than a failure when nothing matches.

## Build links

You can link a source build to a target build to express dependencies between projects:

```bash
yontrack build link \
    --from-project <project> --from-build <build> \
    --to-project <other-project> --to-build <other-build>
```

Each build (source and target) can be identified in three ways:

* by project name and build name:

```bash
yontrack build link \
    --from-project proj-a --from-build 42 \
    --to-project proj-b --to-build 7
```

* by project name and version (release label):

```bash
yontrack build link \
    --from-project proj-a --from-version 2.0.0 \
    --to-project proj-b --to-version 1.2.3
```

* by build ID:

```bash
yontrack build link --from-id 101 --to-id 202
```

These can be mixed freely. An optional `--qualifier` (or `-q`) can be provided to qualify the nature of the link:

```bash
yontrack build link \
    --from-project proj-a --from-build 42 \
    --to-project proj-b --to-version 1.2.3 \
    --qualifier integration
```

The `--from-project` flag defaults to the `YONTRACK_PROJECT_NAME` environment variable, and `--from-build` defaults to `YONTRACK_BUILD_NAME`, so in a typical CI context you can simply write:

```bash
yontrack build link --to-project proj-b --to-version 1.2.3
```

## Removing build links

To remove dependency links from a build, use `build unlink`. The source build is identified the same way as in `build link` (by name, version, or ID), with the same `YONTRACK_PROJECT_NAME` / `YONTRACK_BUILD_NAME` environment variable defaults.

Remove all links from a build:

```bash
yontrack build unlink --from-project proj-a --from-build 42
```

Remove all links targeting a specific project:

```bash
yontrack build unlink --from-project proj-a --from-build 42 --to-project proj-b
```

Remove a specific qualified link:

```bash
yontrack build unlink --from-project proj-a --from-build 42 --to-project proj-b --qualifier integration
```

Note: `--qualifier` requires `--to-project` to be specified.

## Git integration

Yontrack can leverage SCM information stored in its model, in order to compute change logs or to allow searches based on commits.

For example, to associate a project with a GitHub repository:

```bash
# GitHub setup of the project
yontrack project set-property --project <project> github \
    --configuration github.com \
    --repository yontrack/yontrack \
    --indexation 30 \
    --issue-service self
```

This command associates the project with the `yontrack/yontrack` repository, using the credentials defined by the `github.com` GitHub configuration stored in Yontrack. Additionally, Yontrack will index the content of this repository every `30` minutes and the GitHub issues will be used to track issues.

Whenever a branch is created, you associate it with the corresponding Git branch this way:

```bash
# Git setup of the branch
yontrack branch set-property --project <project> --branch <branch> git \
    --git-branch <branch>
```

> Note that pull requests are also supported. In this case, the `--git-branch` must be something like `PR-123`.

Finally, each build can be associated with a Git commit:

```bash
# Git setup of the build
yontrack build set-property --project <project> --branch <branch> --build <build> git-commit \
    --commit <full commit hash>
```

> Note that the Git commit property on a build can be set directly using the `build setup` command and the `--commit` option:

```bash
yontrack build setup --project <project> --branch <branch> --build <build> --commit <commit>
```

# Validation

One of the most important point of Yontrack is to record _validations_:

```bash
yontrack validate --project <project> --branch <branch> --build <build> --validation <validation> --status <status>
```

where `<status>` is a Yontrack validation run status like `PASSED`, `WARNING` or `FAILED`.

## Data validation

Additionally, a validation run can be created with some
[data](https://static.nemerosa.net/ontrack/release/latest/docs/doc/index.html#validation-stamps-data). For example, to create a test summary validation:

```bash
yontrack validate --project <project> --branch <branch> --build <build> --validation <validation> \
    --data-type net.nemerosa.ontrack.extension.general.validation.TestSummaryValidationDataType \
    --data {passed: 1, skipped: 2, failed: 3}
```

The later syntax is pretty cumbersome and the CLI provides dedicated commands for the most used data types:

* for CHML data type:

```bash
yontrack validate --project <project> --branch <branch> --build <build> --validation <validation> \
    chml \
        --critical 0 \
        --high 2 \
        --medium 25 \
        --low 1214
```

* for test summary data type:

```bash
yontrack validate --project <project> --branch <branch> --build <build> --validation <validation> \
    tests \
        --passed 20 \
        --skipped 2 \
        --failed 1
```

* for percentage data type:

```bash
yontrack validate --project <project> --branch <branch> --build <build> --validation <validation> \
    percentage \
        --value 87
```

* for metrics data type:

```bash
yontrack validate --project <project> --branch <branch> --build <build> --validation <validation> \
    metrics \
        --metric speed=1.5 \
        --metric acceleration=0.25 \
        --metrics weight=145,height=185.1
```

## Run info

The `validate` commands accept additional flags to set the run info on a validation (source & trigger, duration):

* `--run-time` - duration of the validation in seconds
* `--source-type` - type of source for the validation, for example, the CI name like `jenkins`
* `--source-uri` - the URI to the source of the validation, for example, the URL to a Jenkins job
* `--trigger-type` - how the validation was triggered (for example: `scm`)
* `--trigger-data` - data associated with the trigger (for example, a Git commit)

For example, to set the validation duration on a test summary validation:

```bash
yontrack validate --project <project> --branch <branch> --build <build> --validation <validation> \
    --run-time 80 \
    tests \
        --passed 20 \
        --skipped 2 \
        --failed 1
```

# Auto-versioning

The Yontrack CLI can be used to set up the auto-versioning configuration for a branch.

Given the `auto-versioning.yaml` file containing the configuration, the call looks like:

```bash
yontrack branch --project <project> --branch <branch> auto-versioning \
    --yaml auto-versioning.yaml
```

The `auto-versioning.yaml` file looks like:

```yaml
dependencies:
  - sourceProject: my-library
    sourceBranch: release-1.3
    sourcePromotion: IRON
    targetPath: gradle.properties
    targetProperty: my-version
    postProcessing: jenkins
    postProcessingConfig:
      dockerImage  : openjdk:8
      dockerCommand: ./gradlew clean
```

> The format of this file is fully described in the Yontrack documentation at
> https://docs.yontrack.com/yontrack/ref/latest/content/integrations/auto-versioning/auto-versioning.html

In a parent repository, you can use the auto-versioning check to automatically create the dependency links.

```yaml
yontrack build auto-versioning-check \
  --project <project> \
  --branch <branch> \
  --build <build>
```

# Notifications

The CLI can be used to setup subscriptions on some entities.

For example, to send a Slack message each time a promotion is granted:

```shell
yontrack promotion subscribe \
  --project <project> \
  --branch <branch> \
  --promotion BRONZE \
  --name "My subscription" \
  slack \
  --channel "#test" \
  --type "SUCCESS"
```

You can also use generic notifications. The code below is equivalent to the one above:

```shell
yontrack promotion subscribe \
  --project <project> \
  --branch <branch> \
  --promotion BRONZE \
  --name "My subscription" \
  generic \
  --channel slack \
  --channel-config '{"channel":"#test","type":"SUCCESS"}'
```

You can also use a specific content template by using the `--template` argument:

```shell
yontrack promotion subscribe \
  --project <project> \
  --branch <branch> \
  --promotion BRONZE \
  --name "My subscription" \
  slack \
  --channel "#test" \
  --type "SUCCESS" \
  --template 'Build ${build} has been promoted to ${promotionLevel}. Well done :)'
```

# Deployments

When [environments and slots](https://docs.yontrack.com/yontrack/ref/latest/content/integrations/environments/environments.html) are configured, a build can be deployed into an environment from the command line.

## Getting a slot

A slot is where a project is deployed into an environment. To get the one for a project:

```bash
yontrack slot get --project my-project --environment production
```

By default this prints the slot ID. The slot also knows which build it last deployed, which is what tells you whether a deployment is needed at all:

```bash
yontrack slot get --project my-project --environment production --output json
```

```json
{
  "id": "b6b8a2c1-...",
  "lastDeployedPipeline": {
    "id": "2957ff78-...",
    "number": 5,
    "build": {
      "id": "11409",
      "name": "20260901055547-46",
      "displayName": "1.2.3"
    }
  }
}
```

`--output env` prints the same information as shell exports, for use in a CI script:

```bash
eval "$(yontrack slot get --project my-project --environment production --output env)"
echo "$YONTRACK_SLOT_ID $YONTRACK_SLOT_DEPLOYED_BUILD_NAME"
```

A slot which has never completed a deployment has no last deployed build, and those values are empty.

## Starting a deployment

To deploy a build into a slot, start a pipeline for it. The build is identified by name:

```bash
yontrack slot pipeline start --project my-project --environment production --build 42
```

or by version - its release label:

```bash
yontrack slot pipeline start --project my-project --environment production --version 1.2.3
```

The command prints the ID of the pipeline it started. Starting a pipeline is only the beginning of a deployment: the slot's own workflows carry it the rest of the way, so keep this ID to follow what happens next.

```bash
yontrack slot pipeline start --project my-project --environment production --build 42 --output json
```

```json
{
  "id": "2957ff78-...",
  "number": 5
}
```

The command fails if the deployment is refused - because the build does not meet the slot's admission rules, for example - so a CI job does not have to inspect the output to know whether it worked.

# Misc

## Direct GraphQL calls

The Yontrack CLI uses the GraphQL API of Yontrack for its communication. The `graphql` command allows to run raw GraphQL queries.

For example:

```bash
yontrack graphql \
    --query 'query ProjectList($name: String!) { projects(name: $name) { id name branches { name } } }' \
    --var name=yontrack
```

`--var` always sends a string. For a variable of any other type - a number, a boolean, a nested input object - pass the whole variables object as JSON instead:

```bash
yontrack graphql \
    --query 'mutation Start($input: StartSlotPipelineInput!) { startSlotPipeline(input: $input) { errors { message } } }' \
    --vars-json '{"input": {"slotId": "b6b8a2c1-...", "buildId": 11409}}'
```

Both can be used together; `--var` is applied last, which is handy for overriding a single string in an otherwise fixed object.

Yontrack reports a refused mutation inside the response, as an `errors` list on the payload, rather than as an HTTP or GraphQL error. By default `graphql` prints that response and succeeds, leaving the check to the caller. Use `--fail-on-user-errors` to have the command fail instead:

```bash
yontrack graphql --fail-on-user-errors \
    --query 'mutation { ... }'
```

The response is still printed before the command fails.

## General options

The `--graphqh-log` flag is available for all commands, to enable some tracing on the console for the GraphQL requests and responses.

# Integrations

While the Yontrack CLI can be used directly, there are direct integrations in some environments.

## Jenkins

The [`ontrack-jenkins-cli-pipeline`](https://github.com/nemerosa/ontrack-jenkins-cli-pipeline/) Jenkins pipeline library allows an easy integration between your `Jenkinsfile` pipelines and Yontrack.

## GitHub actions

* [`ontrack-github-actions-cli-setup`](https://github.com/nemerosa/ontrack-github-actions-cli-setup) - _installation of the CLI and simplified GitHub/Git setup_
* [`ontrack-github-actions-cli-validation`](https://github.com/nemerosa/ontrack-github-actions-cli-validation) - _creation of validation runs based on GitHub workflow information_

# Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
