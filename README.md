Yontrack CLI
============

[![Build](https://github.com/yontrack/yontrack-cli/actions/workflows/go.yml/badge.svg)](https://github.com/yontrack/yontrack-cli/actions/workflows/go.yml)

[Yontrack](https://github.com/nemerosa/ontrack) is an application which store all events which happen in your CI/CD environment: branches, builds, validations, promotions, labels, commits, etc. It allows your delivery chains to reach new levels by driving your pipelines using real-time data.

The Yontrack CLI is a Command Line Interface tool, available on many platforms, which allows you to feed information into Yontrack from any shell platform.

> Yontrack was previously called Ontrack. The GitHub organisation, the companion repositories and the GraphQL schema file still carry the old name; links and commands below use it deliberately and are not stale.

> The Yontrack CLI 5.x works with [Yontrack](https://github.com/nemerosa/ontrack) 5.x.

# Installation

## Homebrew

On macOS, and on Linux where Homebrew is installed:

```bash
brew install yontrack/tap/yontrack
```

After that, `brew upgrade` carries the CLI along with everything else Homebrew manages, which is the reason to prefer this route on macOS.

It installs the very binary the release publishes, pinned to the same sha256 the installer below verifies, so the two routes deliver identical bytes. The tap ships a *formula* rather than a cask on purpose: Homebrew quarantines casks and does not quarantine formulae, so this route never meets the Gatekeeper dialog described at the end of this section.

Homebrew 6 asks that third-party taps be trusted before it loads them. If it says `yontrack/tap` is not trusted:

```bash
brew trust --formula yontrack/tap/yontrack
```

On an Apple Silicon Mac this pours a prebuilt bottle, so it needs nothing from Xcode and takes a second or two.

**On an Intel Mac** there is no bottle, and Homebrew runs its build-from-source checks on any formula without one — so it insists on current Command Line Tools even though nothing here is compiled. Installing Homebrew installs them, so this normally passes unnoticed; it bites after a macOS upgrade that leaves them behind. If `brew install` stops with *"Your Command Line Tools are too outdated"*, that check is what stopped it:

```bash
sudo rm -rf /Library/Developer/CommandLineTools
sudo xcode-select --install
```

Linux has no equivalent requirement. The install script below has none either, and is the way in if you would rather not.

## The install script

On macOS and Linux, with or without Homebrew:

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
found in the CI context - mostly, the environment variables provided by the CI engine: Jenkins, GitHub Actions,
Bitbucket Pipelines or GitLab CI.

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
and its configuration, as JSON. For example, to create a CHML validation type:

```bash
yontrack validation-stamp setup --project <project> --branch <branch> --validation <validation> \
    --data-type net.nemerosa.ontrack.extension.general.validation.CHMLValidationDataType \
    --data-config '{"warningLevel":"HIGH","warningValue":1,"failedLevel":"CRITICAL","failedValue":1}'
```

The configuration is given in the data type's _form_ shape, which is what Yontrack reads when setting a validation
stamp up - not the shape it stores. For CHML and security findings, that's the flat `warningLevel` / `warningValue` /
`failedLevel` / `failedValue` of `.yontrack/ci.yaml`, not nested `{level, value}` objects. A `--data-config` which is
not valid JSON is refused before anything is sent.

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

For a Bitbucket Cloud repository:

```bash
# Bitbucket Cloud setup of the project
yontrack project set-property --project <project> bitbucket-cloud \
    --configuration bitbucket-cloud \
    --workspace my-workspace \
    --repository my-repository \
    --indexation 30 \
    --issue-service jira//my-jira
```

The `--workspace` option is the Bitbucket Cloud workspace slug the repository belongs to; the `bitbucket-cloud` configuration stored in Yontrack only holds the credentials, and can therefore be shared by projects in different workspaces.

For a GitLab repository:

```bash
# GitLab setup of the project
yontrack project set-property --project <project> gitlab \
    --configuration gitlab.com \
    --repository my-group/my-subgroup/my-project \
    --indexation 30 \
    --issue-service self
```

The `--repository` option is the full GitLab project path. Subgroups are normal on GitLab, so this path can be arbitrarily deep: both `group/project` and `group/subgroup/project` are given as-is.

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

* for number data type:

```bash
yontrack validate --project <project> --branch <branch> --build <build> --validation <validation> \
    number \
        --value 0
```

> The `--value` flag is required. For a validation stamp counting issues, where `0` means "nothing
> found", omitting the value would otherwise record a clean result instead of failing.

* for metrics data type:

```bash
yontrack validate --project <project> --branch <branch> --build <build> --validation <validation> \
    metrics \
        --metric speed=1.5 \
        --metric acceleration=0.25 \
        --metrics weight=145,height=185.1
```

## Security findings

A report of security findings is sent with the `findings` subcommand. The CLI sends the file as is:
Yontrack parses it, records the findings and computes the status of the validation from the
thresholds of its `security-findings` validation stamp.

> This command requires Yontrack 6.0, unlike the rest of this CLI. Against Yontrack 5.x, it fails
> with Yontrack's own error.

```bash
yontrack validate --project <project> --branch <branch> --build <build> --validation <validation> \
    findings \
        --format trivy \
        --kind IMAGE \
        --report trivy.json
```

* `--format` (required) - format of the report:
  * `findings` - Yontrack's neutral format
  * `sarif` - SARIF 2.1
  * `trivy` - Trivy JSON (vulnerabilities only)
* `--kind` (required) - what was scanned: `IMAGE`, `CODE`, `SECRETS`, `DAST`, `DEPENDENCIES` or `OTHER`
* `--scanner` - name of the scanner, overriding the one Yontrack reads from the report
* `--report` (required) - path to the report, a JSON file

> `sarif` and `trivy` rely on a licensed feature of Yontrack; `findings` does not.

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

### Run info defaults in CI

When the CLI recognises the CI engine it runs in, the flags which are _not_ set explicitly
are defaulted from the environment of that engine, as described in the sections below.

Flags set on the command line always take precedence, and nothing is defaulted outside of a
recognised CI engine. The run time is never guessed: only the caller knows how long the run
took. A value which is not available - because the variable it comes from is not set - is
left empty rather than guessed.

The engines are tried in the order of the sections below and the first one recognised wins;
that only matters in the unlikely case where the variables of two engines are both present.

#### Bitbucket Pipelines defaults

Detected using the `BITBUCKET_BUILD_NUMBER` environment variable.

| Flag | Default |
|---|---|
| `--source-type` | `bitbucket-pipeline` |
| `--source-uri` | `https://bitbucket.org/$BITBUCKET_WORKSPACE/$BITBUCKET_REPO_SLUG/pipelines/results/$BITBUCKET_BUILD_NUMBER` |
| `--trigger-type` | `commit` |
| `--trigger-data` | `$BITBUCKET_COMMIT` |

> See [Bitbucket Pipelines](#bitbucket-pipelines) for the pipeline these defaults are meant for.

#### GitLab CI defaults

Detected using the `GITLAB_CI` environment variable, which GitLab sets to `true` in every job.

| Flag | Default |
|---|---|
| `--source-type` | `gitlab-pipeline` |
| `--source-uri` | `$CI_PIPELINE_URL` |
| `--trigger-type` | `commit` |
| `--trigger-data` | `$CI_COMMIT_SHA` |

> See [GitLab CI](#gitlab-ci) for the pipeline these defaults are meant for.

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

## Bitbucket Pipelines

There is no Yontrack pipe or app in the Bitbucket marketplace: in Bitbucket Pipelines, the CLI is installed and driven
directly. Two traits of Pipelines shape how that is done:

* each step runs in its own container, so nothing a step installs, configures or exports survives into the next one;
* the only thing handed from a step to the next ones is the files it declares as `artifacts`.

So the CLI is installed again in every step which talks to Yontrack, and the build registered at the start of the
pipeline is carried to the later steps in a file.

> Independently of the pipeline, and only once, associate the Yontrack project with its Bitbucket Cloud repository so
> that change logs and commit searches work. That is `project set-property bitbucket-cloud`, which needs the workspace
> slug in `--workspace` - see [Git integration](#git-integration).

### Credentials

Define these [repository or workspace variables](https://support.atlassian.com/bitbucket-cloud/docs/variables-and-secrets/).
The first two carry credentials and must be marked as _secured_:

| Variable | |
|---|---|
| `YONTRACK_URL` | URL of the Yontrack installation, like `https://yontrack.example.com` |
| `YONTRACK_TOKEN` | A Yontrack API token |
| `YONTRACK_CLI_VERSION` | The CLI release to install, like `5.4.0`. Not a secret. |

These are the variables the scripts below hand to `config create`; the CLI does not read them itself. The only
environment variables it reads are `YONTRACK_PROJECT_NAME`, `YONTRACK_BRANCH_NAME`, `YONTRACK_BUILD_NAME` and
`YONTRACK_BUILD_ID`, all of them produced by `--output env`.

### Installing the CLI

Every step which uses the CLI starts with the same three lines:

```bash
export PATH="$HOME/.local/bin:$PATH"
curl -fsSL https://raw.githubusercontent.com/yontrack/yontrack-cli/main/install.sh | VERSION="$YONTRACK_CLI_VERSION" sh
yontrack config create prod "$YONTRACK_URL" --token "$YONTRACK_TOKEN"
```

The [install script](#the-install-script) needs `curl` in the image and installs into `$HOME/.local/bin`, which is why
the `PATH` is extended first. `VERSION` pins the release: with `YONTRACK_CLI_VERSION` left undefined the script installs
the latest one, and the pipeline then depends on whatever was published last.

Keep the three lines in a single `- |` block, so that the `export` and the commands which need it are run by the same
shell. The configuration `config create` writes carries the token - leave it out of the `artifacts` patterns below.

### Registering the build

`ci config` creates the project, the branch and the build from the environment. The Bitbucket variables are passed with
`--env-all`, which selects every variable carrying a given prefix:

```bash
yontrack ci config \
  --env-all BITBUCKET_ \
  --var version="1.0.$BITBUCKET_BUILD_NUMBER" \
  --output env > yontrack.env
```

Yontrack recognises Bitbucket Pipelines and Bitbucket Cloud from those variables on its own, so neither
`--ci bitbucket-pipelines` nor `--scm bitbucket-cloud` has to be given; the flags are there to force them.

`--var version=...` builds the version from `BITBUCKET_BUILD_NUMBER`, the number Bitbucket increments at each run of the
pipeline. It is used from `.yontrack/ci.yaml` as:

```yaml
version: v1
configuration:
  build:
    name: "{{ vars \"version\" }}"
```

Only the `export` lines go to standard output - `ci config` traces what it reads on standard error - so the redirection
above captures the identity of the build and nothing else.

> `--env-all` sends every variable carrying the prefix to Yontrack and echoes each of them to the build log. If a step
> defines a sensitive `BITBUCKET_` variable of its own, list the ones you need with `--env` instead.

### Passing the build between steps

`--output env` wrote the identity of the build into `yontrack.env`:

```text
export YONTRACK_PROJECT_ID=123
export YONTRACK_PROJECT_NAME=my-project
export YONTRACK_BRANCH_ID=456
export YONTRACK_BRANCH_NAME=main
export YONTRACK_BUILD_ID=789
export YONTRACK_BUILD_NAME=1.0.42
```

Declare that file as an artifact of the step which produced it:

```yaml
        artifacts:
          - yontrack.env
```

Bitbucket then makes it available to every later step, where sourcing it is enough for the other commands to find the
build without any `--project`, `--branch` or `--build` flag:

```bash
source yontrack.env
yontrack validate --validation unit-tests --status PASSED
```

**This is the route to prefer**: one call to Yontrack for the whole pipeline, and it does not depend on what the build
carries.

The other route is to look the build up again in each step, from the commit the pipeline runs on:

```bash
eval "$(yontrack build search --project my-project --commit "$BITBUCKET_COMMIT" --count 1 --output env)"
```

This prints the very same exports, but it costs a query per step, the project name has to be known by every step, and it
only finds the build if that build carries the Git commit property - which `build setup --commit "$BITBUCKET_COMMIT"`
sets. Add `--accept-not-found` if a missing build should not fail the search itself - nothing is then exported, and the
failure moves to the first command needing `YONTRACK_PROJECT_NAME`.

### Validations

`after-script` runs once the `script` of its step is over, whether it succeeded or not, and Bitbucket sets
`BITBUCKET_EXIT_CODE` to the exit code of that script. That is where a validation belongs: recorded from `after-script`,
a failing step gets a `FAILED` validation run instead of being skipped along with the step.

```yaml
      after-script:
        - |
          export PATH="$HOME/.local/bin:$PATH"
          source yontrack.env
          if [ "$BITBUCKET_EXIT_CODE" = "0" ]; then
            yontrack validate --validation unit-tests --status PASSED
          else
            yontrack validate --validation unit-tests --status FAILED
          fi
```

The `PATH` is extended again because `after-script` runs in a shell of its own; the binary itself is still there,
installed by the `script` of the same step.

### Run info

Nothing has to be passed for the run info: inside Bitbucket Pipelines the CLI fills in the source type, the pipeline
URL, the trigger type and the commit on its own - see
[Bitbucket Pipelines defaults](#bitbucket-pipelines-defaults) for the exact values. The run time
is the exception, since only the caller knows it:

```bash
start=$(date +%s)
./gradlew test
yontrack validate --validation unit-tests --status PASSED --run-time $(( $(date +%s) - start ))
```

Measured in the `script` like this, the duration does not reach the `after-script`, which runs in a shell of its own:
a step which wants both the duration and `BITBUCKET_EXIT_CODE` writes the start time to a file first.

### A complete `bitbucket-pipelines.yml`

```yaml
image: atlassian/default-image:4

pipelines:
  default:
    - step:
        name: Yontrack build
        script:
          - |
            export PATH="$HOME/.local/bin:$PATH"
            curl -fsSL https://raw.githubusercontent.com/yontrack/yontrack-cli/main/install.sh | VERSION="$YONTRACK_CLI_VERSION" sh
            yontrack config create prod "$YONTRACK_URL" --token "$YONTRACK_TOKEN"
            yontrack ci config \
              --env-all BITBUCKET_ \
              --var version="1.0.$BITBUCKET_BUILD_NUMBER" \
              --output env > yontrack.env
        artifacts:
          - yontrack.env
    - step:
        name: Unit tests
        script:
          - |
            export PATH="$HOME/.local/bin:$PATH"
            curl -fsSL https://raw.githubusercontent.com/yontrack/yontrack-cli/main/install.sh | VERSION="$YONTRACK_CLI_VERSION" sh
            yontrack config create prod "$YONTRACK_URL" --token "$YONTRACK_TOKEN"
          - ./gradlew test
        after-script:
          - |
            export PATH="$HOME/.local/bin:$PATH"
            source yontrack.env
            if [ "$BITBUCKET_EXIT_CODE" = "0" ]; then
              yontrack validate --validation unit-tests --status PASSED
            else
              yontrack validate --validation unit-tests --status FAILED
            fi
```

The first step registers the build and publishes `yontrack.env`; the second one runs the tests and records their
outcome against that build. Any further step follows the same shape: install, configure, `source yontrack.env`, then
whichever `yontrack` command it needs.

## GitLab CI

There is no Yontrack component in the GitLab CI/CD catalog: in GitLab CI, the CLI is installed and driven directly.
Two traits of GitLab CI shape how that is done:

* each job runs on its own runner, in a container of its own, so nothing a job installs, configures or exports
  survives into the next one;
* what a job hands to the later ones is what it declares under `artifacts` - files, given as `paths`, or variables,
  given as a `dotenv` report.

So the CLI is installed again in every job which talks to Yontrack, and the build registered at the start of the
pipeline is carried to the later jobs in a file.

> Independently of the pipeline, and only once, associate the Yontrack project with its GitLab project so that change
> logs and commit searches work. That is `project set-property gitlab`, whose `--repository` is the full project path,
> as deep as the subgroups go - see [Git integration](#git-integration).

### Credentials

Define these [CI/CD variables](https://docs.gitlab.com/ci/variables/) on the project, or on the group when several
projects feed the same Yontrack. The first two carry credentials and the token must be _masked_:

| Variable | |
|---|---|
| `YONTRACK_URL` | URL of the Yontrack installation, like `https://yontrack.example.com` |
| `YONTRACK_TOKEN` | A Yontrack API token. Masked, so that it is hidden in the job logs. |
| `YONTRACK_CLI_VERSION` | The CLI release to install, like `5.4.0`. Not a secret. |

_Protected_ is the setting to weigh: a protected variable reaches only the pipelines running on a protected branch or
tag, and every other pipeline gets an empty `YONTRACK_TOKEN` and fails to register its build. Protect the token when
only protected branches feed Yontrack; leave it unprotected when feature branches do too.

These are the variables the scripts below hand to `config create`; the CLI does not read them itself. The only
environment variables it reads are `YONTRACK_PROJECT_NAME`, `YONTRACK_BRANCH_NAME`, `YONTRACK_BUILD_NAME` and
`YONTRACK_BUILD_ID`, all of them produced by `--output env`.

### Installing the CLI

Every job which uses the CLI starts with the same lines, which belong in its `before_script`:

```bash
apt-get update -qq && apt-get install -y -qq --no-install-recommends curl ca-certificates
export PATH="$HOME/.local/bin:$PATH"
curl -fsSL https://raw.githubusercontent.com/yontrack/yontrack-cli/main/install.sh | VERSION="$YONTRACK_CLI_VERSION" sh
yontrack config create prod "$YONTRACK_URL" --token "$YONTRACK_TOKEN"
```

The [install script](#the-install-script) needs `curl` in the image, which the first line installs the Debian and
Ubuntu way; an Alpine image uses `apk add --no-cache curl` instead, and an image which already carries `curl` drops
the line altogether. The script installs into `$HOME/.local/bin`, which is why the `PATH` is extended next. `VERSION`
pins the release: with `YONTRACK_CLI_VERSION` left undefined the script installs the latest one, and the pipeline then
depends on whatever was published last.

`before_script` and `script` run in the same shell, so that `export` reaches every command of the job - `after_script`
does not, and extends the `PATH` again. The configuration `config create` writes is `.yontrack-config.yaml` in the
working directory, which is the clone of the repository: it carries the token, so keep it out of the `artifacts` and
`cache` patterns of the job.

### Registering the build

`ci config` creates the project, the branch and the build from the environment. The GitLab variables are passed with
`--env-all`, which selects every variable carrying a given prefix:

```bash
yontrack ci config \
  --env-all CI_ \
  --env-all GITLAB_CI \
  --var version="1.0.$CI_PIPELINE_IID" \
  --output env > yontrack.env
```

The flag is given twice because GitLab's predefined variables are the `CI_` ones, while the variable the CI engine is
recognised by is `GITLAB_CI`, which carries no such prefix - a prefix matches its own variable, so the second
`--env-all` selects that one.

Yontrack recognises GitLab CI and GitLab from those variables on its own, so neither `--ci gitlab-ci` nor
`--scm gitlab` has to be given; the flags are there to force them.

`--var version=...` builds the version from `CI_PIPELINE_IID`, the counter GitLab increments at each pipeline of the
project. Its sibling `CI_PIPELINE_ID` is unique across the whole GitLab instance and jumps by arbitrary amounts, which
makes a poor version number. The variable is used from `.yontrack/ci.yaml` as:

```yaml
version: v1
configuration:
  build:
    name: "{{ vars \"version\" }}"
```

Only the `export` lines go to standard output - `ci config` traces what it reads on standard error - so the
redirection above captures the identity of the build and nothing else.

> `--env-all` sends every variable carrying the prefix to Yontrack and echoes each of them to the job log. On GitLab
> the `CI_` prefix covers more than the identity of the pipeline: `CI_JOB_TOKEN`, `CI_REPOSITORY_URL`, which embeds
> it, and `CI_REGISTRY_PASSWORD` where the container registry is enabled, are all predefined and all carry
> credentials. Where that matters, list the variables you need with `--env` instead.

### Passing the build between jobs

`--output env` wrote the identity of the build into `yontrack.env`:

```text
export YONTRACK_PROJECT_ID=123
export YONTRACK_PROJECT_NAME=my-project
export YONTRACK_BRANCH_ID=456
export YONTRACK_BRANCH_NAME=main
export YONTRACK_BUILD_ID=789
export YONTRACK_BUILD_NAME=1.0.42
```

Declare that file as an artifact of the job which produced it:

```yaml
  artifacts:
    paths:
      - yontrack.env
```

A job is given the artifacts of every job of the earlier stages - or of the jobs it lists in `needs` - so sourcing the
file is enough for the other commands to find the build without any `--project`, `--branch` or `--build` flag:

```bash
. ./yontrack.env
yontrack validate --validation unit-tests --status PASSED
```

`.` rather than `source`, which is a `bash` builtin that the `/bin/sh` of a minimal image does not have.

**This is the route to prefer**: one call to Yontrack for the whole pipeline, and it does not depend on what the build
carries.

If sourcing a file in each job is one step too many, the same file can be declared as a `dotenv` report instead, and
GitLab turns it into variables of the later jobs. The report is read as `KEY=value` lines and does not accept the
`export ` prefix that `--output env` writes, so that prefix is stripped first:

```bash
yontrack ci config \
  --env-all CI_ \
  --env-all GITLAB_CI \
  --var version="1.0.$CI_PIPELINE_IID" \
  --output env | sed 's/^export //' > yontrack.env
```

```yaml
  artifacts:
    reports:
      dotenv: yontrack.env
```

The other route is to look the build up again in each job, from the commit the pipeline runs on:

```bash
eval "$(yontrack build search --project my-project --commit "$CI_COMMIT_SHA" --count 1 --output env)"
```

This prints the very same exports, but it costs a query per job, the project name has to be known by every job, and it
only finds the build if that build carries the Git commit property - which `build setup --commit "$CI_COMMIT_SHA"`
sets. Add `--accept-not-found` if a missing build should not fail the search itself - nothing is then exported, and
the failure moves to the first command needing `YONTRACK_PROJECT_NAME`. In a merged results pipeline it finds nothing
at all: `CI_COMMIT_SHA` is then the temporary merge commit GitLab created for the run, and not the commit the build
was registered with.

### Validations

`after_script` runs once the `script` of its job is over, whether it succeeded or not, and GitLab sets `CI_JOB_STATUS`
to `success`, `failed` or `canceled`. That is where a validation belongs: recorded from `after_script`, a failing job
gets a `FAILED` validation run instead of being skipped along with the job.

```yaml
  after_script:
    - |
      export PATH="$HOME/.local/bin:$PATH"
      . ./yontrack.env
      if [ "$CI_JOB_STATUS" = "success" ]; then
        yontrack validate --validation unit-tests --status PASSED
      else
        yontrack validate --validation unit-tests --status FAILED
      fi
```

The `PATH` is extended again because `after_script` runs in a shell of its own, which the `before_script` of the job
does not feed; the binary itself is still there, in the same container. A cancelled job falls into the `else` above
and is recorded as a failure - test `CI_JOB_STATUS` for `failed` alone if that is not what you want.

### Run info

Nothing has to be passed for the run info: inside GitLab CI the CLI fills in the source type, the pipeline URL, the
trigger type and the commit on its own - see [GitLab CI defaults](#gitlab-ci-defaults) for the exact values. The run
time is the exception, since only the caller knows it:

```bash
start=$(date +%s)
./gradlew test
yontrack validate --validation unit-tests --status PASSED --run-time $(( $(date +%s) - start ))
```

Measured in the `script` like this, the duration does not reach the `after_script`, which runs in a shell of its own:
a job which wants both the duration and `CI_JOB_STATUS` writes the start time to a file first.

### A complete `.gitlab-ci.yml`

```yaml
stages:
  - yontrack
  - test

default:
  image: ubuntu:24.04
  before_script:
    - apt-get update -qq && apt-get install -y -qq --no-install-recommends curl ca-certificates
    - export PATH="$HOME/.local/bin:$PATH"
    - curl -fsSL https://raw.githubusercontent.com/yontrack/yontrack-cli/main/install.sh | VERSION="$YONTRACK_CLI_VERSION" sh
    - yontrack config create prod "$YONTRACK_URL" --token "$YONTRACK_TOKEN"

yontrack-build:
  stage: yontrack
  script:
    - |
      yontrack ci config \
        --env-all CI_ \
        --env-all GITLAB_CI \
        --var version="1.0.$CI_PIPELINE_IID" \
        --output env > yontrack.env
  artifacts:
    paths:
      - yontrack.env

unit-tests:
  stage: test
  image: eclipse-temurin:21-jdk
  script:
    - ./gradlew test
  after_script:
    - |
      export PATH="$HOME/.local/bin:$PATH"
      . ./yontrack.env
      if [ "$CI_JOB_STATUS" = "success" ]; then
        yontrack validate --validation unit-tests --status PASSED
      else
        yontrack validate --validation unit-tests --status FAILED
      fi
```

The first job registers the build and publishes `yontrack.env`; the second one runs the tests and records their
outcome against that build. Any further job follows the same shape: the shared `before_script`, `. ./yontrack.env`,
then whichever `yontrack` command it needs.

Both images here are Debian-based, which is what lets one `before_script` serve them; a job on an image of another
family installs `curl` the way that family does, and overrides the `before_script` accordingly.

# Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
