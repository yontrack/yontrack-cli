# Yontrack CLI — Claude guidance

## Project overview

This is `yontrack`, a CLI tool written in Go for feeding data into [Yontrack](https://github.com/nemerosa/ontrack) (a CI/CD traceability platform, formerly called Ontrack — the GitHub organisation still carries the old name). It communicates exclusively via Yontrack's GraphQL API.

## Tech stack

- **Language:** Go
- **CLI framework:** [Cobra](https://github.com/spf13/cobra) (`github.com/spf13/cobra`)
- **HTTP client:** go-resty (`github.com/go-resty/resty/v2`)
- **Config:** Viper + YAML (`~/.yontrack-config.yaml`)
- **GraphQL schema:** `ontrack.graphql` — the full schema of the Ontrack server (reference for available queries/mutations)

## Project structure

```
cmd/           # One file per command or command group
  root.go      # Root cobra command
  build.go     # `build` parent command
  buildSetup.go, buildLink.go, ...  # build subcommands
  branch*.go, validate*.go, ...     # other command groups
utils/
  flagsUtils.go  # Shared flag helpers (GetProjectFlag, GetBuildFlag, etc.)
client/
  ontrackGraphQLClient.go  # GraphQLCall(), CheckDataErrors()
config/
  configService.go  # Config struct, GetSelectedConfiguration()
main.go
ontrack.graphql   # Full Ontrack GraphQL schema (reference only)
```

## Command pattern

Every command follows this structure:

```go
var myCmd = &cobra.Command{
    Use:   "name",
    Short: "...",
    Long:  `...`,
    RunE: func(cmd *cobra.Command, args []string) error {
        return myFunc(cmd)
    },
}

func myFunc(cmd *cobra.Command) error {
    // 1. Read flags
    // 2. Get config: cfg, err := config.GetSelectedConfiguration()
    // 3. Call GraphQL: client.GraphQLCall(cfg, query, variables, &data)
    // 4. Check errors: client.CheckDataErrors(data.SomeMutation.Errors)
    return nil
}

func init() {
    parentCmd.AddCommand(myCmd)
    myCmd.Flags().StringP("flag", "f", "", "description")
}
```

## GraphQL calls

```go
var data struct {
    MutationName struct {
        Errors []struct{ Message string }
    }
}
if err := client.GraphQLCall(cfg, `mutation { ... }`, map[string]interface{}{...}, &data); err != nil {
    return err
}
return client.CheckDataErrors(data.MutationName.Errors)
```

- `client.GraphQLCall` handles auth and HTTP; returns error on HTTP or GraphQL transport errors.
- `client.CheckDataErrors` returns an error if the mutation's `errors` list is non-empty.
- GraphQL `ID` scalars are serialised as `string` in Go structs (e.g. `Id string`).

## Environment variable fallbacks

Many flags fall back to environment variables — replicate this pattern when writing new commands:

| Flag | Env var |
|---|---|
| `--project` | `YONTRACK_PROJECT_NAME` |
| `--branch` | `YONTRACK_BRANCH_NAME` |
| `--build` | `YONTRACK_BUILD_NAME` |

Use `utils.GetProjectFlag(cmd)`, `utils.GetBranchFlag(cmd, ...)`, `utils.GetBuildFlag(cmd)` to handle these automatically.

## Adding a new command

1. Create `cmd/<commandGroup><SubCommand>.go`
2. Declare `var <name>Cmd = &cobra.Command{...}`
3. In `init()`, call `<parent>Cmd.AddCommand(<name>Cmd)` and register flags
4. Look up the relevant mutation/query in `ontrack.graphql` before writing any GraphQL strings

## Build & verify

```bash
go build ./...          # must compile with no errors
go build -o yontrack .  # build the binary
./yontrack <command> --help
```

## Releasing

Releases are cut by pushing a `MAJOR.MINOR.PATCH` tag; `.github/workflows/tag.yml`
does the rest, including generating the changelog from Yontrack. Do not write a
changelog by hand.

The version is chosen from the labels of the issues since the last tag: a
`feature` or `enhancement` makes it a minor, `bug` alone makes it a patch. A new
command or a new flag is additive and still a minor.

**Read [RELEASING.md](RELEASING.md) before cutting one** - it carries the label
check that keeps issues out of the changelog's `Other` group, and how to preview
the changelog before committing to a tag.

## Key conventions

- No branches in `linkBuild`-style mutations — builds are identified project-wide
- Release/version label property type: `net.nemerosa.ontrack.extension.general.ReleasePropertyType`
- Git commit property type: `net.nemerosa.ontrack.extension.git.property.GitCommitPropertyType`
- Branch names are normalised (slashes → dashes) via `utils.NormalizeBranchName`
