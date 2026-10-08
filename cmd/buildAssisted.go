/*
Copyright © 2021 Damien Coraboeuf <damien.coraboeuf@nemerosa.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"yontrack/assistants"
	"yontrack/client"
	config "yontrack/config"
	"yontrack/utils"

	"github.com/spf13/cobra"
)

// buildAssistedCmd represents the build assisted command
var buildAssistedCmd = &cobra.Command{
	Use:   "assisted",
	Short: "Sets whether the change of a build was written with assistants",
	Long: `Sets the assisted change of a build: whether the commits since the previous build
were written with assistants (coding agents), how many, and the links to their
sessions. Requires Yontrack 6.0.

Yontrack computes it when the project has an SCM. Without one, the CI sets it,
either from git:

    yontrack build assisted --project my-project --branch main --build 42 --from-git <previous-commit>..<commit>

or explicitly:

    yontrack build assisted --project my-project --branch main --build 42 \
        --assistants "Claude Code,Codex" --assisted-commits 3 --total-commits 5 \
        --session-link https://claude.ai/code/session_...

--from-git reads the commits of the range with 'git log', in the current
directory, and recognises their assistants exactly as Yontrack does: the
built-in conventions (Co-Authored-By trailers of the agents, Assisted-by,
Claude-Session, the bot names) and the custom patterns of the Agent markers
settings. Every commit of the range counts, merges included. Reading the custom
patterns needs a token allowed to read the global settings: without it, only
the built-in conventions are applied, with a warning.

The CI knows the previous commit, like github.event.before or
CI_COMMIT_BEFORE_SHA: the range is never looked up in Yontrack.

The value is set even when no commit was assisted: that the CI looked and found
none is a fact, while a build without the value counts as assisted.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return buildAssisted(cmd)
	},
}

// explicitAssistedFlags are the flags giving the assisted change explicitly,
// rather than from git.
var explicitAssistedFlags = []string{"assistants", "assisted-commits", "total-commits", "session-link"}

func buildAssisted(cmd *cobra.Command) error {
	project, branch, build, err := utils.GetProjectBranchBuildFlags(cmd, false, true)
	if err != nil {
		return err
	}
	gitRange, err := cmd.Flags().GetString("from-git")
	if err != nil {
		return err
	}

	explicit := false
	for _, flag := range explicitAssistedFlags {
		explicit = explicit || cmd.Flags().Changed(flag)
	}
	switch {
	case gitRange != "" && explicit:
		return errors.New("--from-git is mutually exclusive with --assistants, --assisted-commits, --total-commits and --session-link")
	case gitRange == "" && !explicit:
		return errors.New("one of --from-git or --total-commits is required")
	case explicit && !cmd.Flags().Changed("total-commits"):
		return errors.New("--total-commits is required with --assistants, --assisted-commits and --session-link")
	}

	var commits []assistants.Commit
	if gitRange != "" {
		// Read first: a range which does not resolve fails before any call
		if commits, err = assistants.GitLog("", gitRange); err != nil {
			return err
		}
	}

	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return err
	}

	var change assistants.Change
	if gitRange != "" {
		rules, err := agentMarkerRules(cmd, cfg)
		if err != nil {
			return err
		}
		change = assistants.Summarize(commits, rules)
	} else if change, err = explicitAssistedChange(cmd); err != nil {
		return err
	}

	var data struct {
		SetBuildAssistedChangeProperty struct {
			Errors []struct{ Message string }
		}
	}
	if err := client.GraphQLCall(cfg, `
		mutation SetBuildAssistedChange(
			$project: String!,
			$branch: String!,
			$build: String!,
			$assistants: [String!],
			$assistedCommits: Int,
			$totalCommits: Int,
			$sessionLinks: [String!]
		) {
			setBuildAssistedChangeProperty(input: {
				project: $project,
				branch: $branch,
				build: $build,
				basis: SET_BY_CI,
				assistants: $assistants,
				assistedCommits: $assistedCommits,
				totalCommits: $totalCommits,
				sessionLinks: $sessionLinks
			}) {
				errors {
					message
				}
			}
		}
	`, map[string]interface{}{
		"project":         project,
		"branch":          branch,
		"build":           build,
		"assistants":      change.Assistants,
		"assistedCommits": change.AssistedCommits,
		"totalCommits":    change.TotalCommits,
		"sessionLinks":    change.SessionLinks,
	}, &data); err != nil {
		return err
	}
	if err := client.CheckDataErrors(data.SetBuildAssistedChangeProperty.Errors); err != nil {
		return err
	}

	summary := fmt.Sprintf("Build %s: %d of %d commits assisted", build, change.AssistedCommits, change.TotalCommits)
	if len(change.Assistants) > 0 {
		summary += " (" + strings.Join(change.Assistants, ", ") + ")"
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), summary)
	return nil
}

// explicitAssistedChange is the assisted change given by the flags, the
// assistants trimmed, distinct and sorted, as Yontrack stores them.
func explicitAssistedChange(cmd *cobra.Command) (assistants.Change, error) {
	names, err := cmd.Flags().GetStringSlice("assistants")
	if err != nil {
		return assistants.Change{}, err
	}
	assistedCommits, err := cmd.Flags().GetInt("assisted-commits")
	if err != nil {
		return assistants.Change{}, err
	}
	totalCommits, err := cmd.Flags().GetInt("total-commits")
	if err != nil {
		return assistants.Change{}, err
	}
	sessionLinks, err := cmd.Flags().GetStringArray("session-link")
	if err != nil {
		return assistants.Change{}, err
	}

	change := assistants.Change{
		Assistants:      []string{},
		AssistedCommits: assistedCommits,
		TotalCommits:    totalCommits,
		SessionLinks:    []string{},
	}
	seen := map[string]bool{}
	for _, name := range names {
		if name = strings.TrimSpace(name); name != "" && !seen[name] {
			seen[name] = true
			change.Assistants = append(change.Assistants, name)
		}
	}
	sort.Strings(change.Assistants)
	change.SessionLinks = append(change.SessionLinks, sessionLinks...)
	return change, nil
}

// agentMarkerRules are the rules of the Agent markers settings of Yontrack.
// When they cannot be read for lack of rights, the built-in conventions only
// are applied, with a warning. An invalid pattern is skipped, with a warning.
func agentMarkerRules(cmd *cobra.Command, cfg *config.Config) (*assistants.Rules, error) {
	var data struct {
		Settings struct {
			SettingsById *struct {
				Values json.RawMessage
			}
		}
	}
	err := client.GraphQLCall(cfg, `
		query AgentMarkers {
			settings {
				settingsById(id: "agent-markers") {
					values
				}
			}
		}
	`, nil, &data)

	settings := assistants.DefaultSettings()
	switch {
	case client.IsForbidden(err):
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(),
			"Warning: this token is not allowed to read the agent markers settings of Yontrack: only the built-in conventions are applied, not the custom patterns.")
	case err != nil:
		return nil, err
	case data.Settings.SettingsById == nil:
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(),
			"Warning: Yontrack has no agent markers settings: only the built-in conventions are applied.")
	default:
		if settings, err = assistants.ParseSettings(data.Settings.SettingsById.Values); err != nil {
			return nil, err
		}
	}

	rules, warnings := assistants.NewRules(settings)
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: agent marker pattern skipped. %s\n", warning)
	}
	return rules, nil
}

func init() {
	buildCmd.AddCommand(buildAssistedCmd)

	buildAssistedCmd.Flags().StringP("project", "p", "", "Name of the project (defaults to YONTRACK_PROJECT_NAME)")
	buildAssistedCmd.Flags().StringP("branch", "b", "", "Name of the branch (defaults to YONTRACK_BRANCH_NAME)")
	buildAssistedCmd.Flags().StringP("build", "n", "", "Name of the build (defaults to YONTRACK_BUILD_NAME)")

	buildAssistedCmd.Flags().String("from-git", "", "Range of commits <previous-commit>..<commit> to read from git, in the current directory. Exclusive with the other flags.")

	buildAssistedCmd.Flags().StringSlice("assistants", nil, "Names of the assistants, comma separated. Empty when no commit was assisted.")
	buildAssistedCmd.Flags().Int("assisted-commits", 0, "Number of commits written with an assistant")
	buildAssistedCmd.Flags().Int("total-commits", 0, "Number of commits in the change of the build. Required unless --from-git is used.")
	buildAssistedCmd.Flags().StringArray("session-link", nil, "Link to an agent session behind the commits. Repeatable.")
}
