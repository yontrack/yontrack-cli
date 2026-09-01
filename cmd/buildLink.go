package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"yontrack/client"
	"yontrack/config"
)

const releasePropertyType = "net.nemerosa.ontrack.extension.general.ReleasePropertyType"
const gitCommitPropertyType = "net.nemerosa.ontrack.extension.git.property.GitCommitPropertyType"

var buildLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Links a build to another build",
	Long: `Links a source build to a target build.

Each build (source and target) can be identified by one of:
  - project name + build name:    --from-project PROJECT --from-build BUILD
  - project name + version label: --from-project PROJECT --from-version VERSION
  - build ID:                     --from-id ID

The same three options apply to the target build (--to-project / --to-build,
--to-project / --to-version, --to-id).

For the source build, --from-project defaults to the YONTRACK_PROJECT_NAME
environment variable and --from-build defaults to YONTRACK_BUILD_NAME.

Examples:

  yontrack build link --from-project proj-a --from-build 42 --to-project proj-b --to-build 7
  yontrack build link --from-id 101 --to-id 202
  yontrack build link --to-project proj-b --to-version 1.2.3
  yontrack build link --from-project proj-a --from-version 2.0.0 --to-project proj-b --to-version 1.2.3 --qualifier integration
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return buildLink(cmd)
	},
}

func buildLink(cmd *cobra.Command) error {
	// Read from-side flags
	fromProject, _ := cmd.Flags().GetString("from-project")
	if fromProject == "" {
		fromProject = os.Getenv("YONTRACK_PROJECT_NAME")
	}
	fromBuild, _ := cmd.Flags().GetString("from-build")
	if fromBuild == "" {
		fromBuild = os.Getenv("YONTRACK_BUILD_NAME")
	}
	fromVersion, err := cmd.Flags().GetString("from-version")
	if err != nil {
		return err
	}
	fromID, err := cmd.Flags().GetInt("from-id")
	if err != nil {
		return err
	}

	// Read to-side flags
	toProject, err := cmd.Flags().GetString("to-project")
	if err != nil {
		return err
	}
	toBuild, err := cmd.Flags().GetString("to-build")
	if err != nil {
		return err
	}
	toVersion, err := cmd.Flags().GetString("to-version")
	if err != nil {
		return err
	}
	toID, err := cmd.Flags().GetInt("to-id")
	if err != nil {
		return err
	}

	qualifier, err := cmd.Flags().GetString("qualifier")
	if err != nil {
		return err
	}

	// Validate from-side: exactly one identification method
	fromMethods := countSet(fromID > 0, fromBuild != "", fromVersion != "")
	if fromMethods == 0 {
		return fmt.Errorf("source build must be identified by one of: --from-id, --from-project + --from-build, or --from-project + --from-version")
	}
	if fromMethods > 1 {
		return fmt.Errorf("source build: only one of --from-id, --from-build, or --from-version may be specified at a time")
	}
	if (fromBuild != "" || fromVersion != "") && fromProject == "" {
		return fmt.Errorf("--from-project (or YONTRACK_PROJECT_NAME) is required when using --from-build or --from-version")
	}

	// Validate to-side: exactly one identification method
	toMethods := countSet(toID > 0, toBuild != "", toVersion != "")
	if toMethods == 0 {
		return fmt.Errorf("target build must be identified by one of: --to-id, --to-project + --to-build, or --to-project + --to-version")
	}
	if toMethods > 1 {
		return fmt.Errorf("target build: only one of --to-id, --to-build, or --to-version may be specified at a time")
	}
	if (toBuild != "" || toVersion != "") && toProject == "" {
		return fmt.Errorf("--to-project is required when using --to-build or --to-version")
	}

	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return err
	}

	// Resolve from-side to ID
	resolvedFromID := fromID
	if resolvedFromID == 0 {
		resolvedFromID, err = resolveBuildID(cfg, fromProject, fromBuild, fromVersion)
		if err != nil {
			return fmt.Errorf("source build: %w", err)
		}
	}

	// Resolve to-side to ID
	resolvedToID := toID
	if resolvedToID == 0 {
		resolvedToID, err = resolveBuildID(cfg, toProject, toBuild, toVersion)
		if err != nil {
			return fmt.Errorf("target build: %w", err)
		}
	}

	// Call linkBuildById
	var data struct {
		LinkBuildById struct {
			Errors []struct {
				Message string
			}
		}
	}
	if err := client.GraphQLCall(cfg, `
		mutation LinkBuildById(
			$fromBuild: Int!,
			$toBuild: Int!,
			$qualifier: String
		) {
			linkBuildById(input: {
				fromBuild: $fromBuild,
				toBuild: $toBuild,
				qualifier: $qualifier
			}) {
				errors {
					message
				}
			}
		}
	`, map[string]interface{}{
		"fromBuild": resolvedFromID,
		"toBuild":   resolvedToID,
		"qualifier": qualifier,
	}, &data); err != nil {
		return err
	}

	return client.CheckDataErrors(data.LinkBuildById.Errors)
}

// buildSearchFilter builds the BuildSearchForm filter map for a build lookup by name or version.
// Exactly one of name or version must be non-empty.
func buildSearchFilter(name, version string) map[string]interface{} {
	filter := map[string]interface{}{"maximumCount": 1}
	if name != "" {
		filter["buildName"] = name
		filter["buildExactMatch"] = true
	} else {
		filter["property"] = releasePropertyType
		filter["propertyValue"] = version
	}
	return filter
}

// resolveBuildID resolves a build to its integer ID using either name or version.
// Exactly one of name or version must be non-empty.
func resolveBuildID(cfg *config.Config, project, name, version string) (int, error) {
	filter := buildSearchFilter(name, version)

	var data struct {
		Builds []struct {
			Id string
		}
	}
	if err := client.GraphQLCall(cfg, `
		query ResolveBuildID($project: String!, $filter: BuildSearchForm!) {
			builds(project: $project, buildProjectFilter: $filter) {
				id
			}
		}
	`, map[string]interface{}{
		"project": project,
		"filter":  filter,
	}, &data); err != nil {
		return 0, err
	}

	if len(data.Builds) == 0 {
		if name != "" {
			return 0, fmt.Errorf("no build found with name %q in project %q", name, project)
		}
		return 0, fmt.Errorf("no build found with version %q in project %q", version, project)
	}
	id, err := strconv.Atoi(data.Builds[0].Id)
	if err != nil {
		return 0, fmt.Errorf("invalid build ID %q returned by server", data.Builds[0].Id)
	}
	return id, nil
}

func countSet(values ...bool) int {
	count := 0
	for _, v := range values {
		if v {
			count++
		}
	}
	return count
}

func init() {
	buildCmd.AddCommand(buildLinkCmd)

	// From-side flags
	buildLinkCmd.Flags().String("from-project", "", "Project of the source build (defaults to YONTRACK_PROJECT_NAME)")
	buildLinkCmd.Flags().String("from-build", "", "Name of the source build (defaults to YONTRACK_BUILD_NAME)")
	buildLinkCmd.Flags().String("from-version", "", "Version (release label) of the source build")
	buildLinkCmd.Flags().Int("from-id", 0, "ID of the source build")

	// To-side flags
	buildLinkCmd.Flags().String("to-project", "", "Project of the target build")
	buildLinkCmd.Flags().String("to-build", "", "Name of the target build")
	buildLinkCmd.Flags().String("to-version", "", "Version (release label) of the target build")
	buildLinkCmd.Flags().Int("to-id", 0, "ID of the target build")

	// Optional qualifier
	buildLinkCmd.Flags().StringP("qualifier", "q", "", "Optional qualifier for the link")
}
