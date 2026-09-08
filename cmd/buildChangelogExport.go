package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"yontrack/client"
	"yontrack/config"
	"yontrack/utils"

	"github.com/spf13/cobra"
)

// buildChangelogExportCmd represents the buildChangelog command
var buildChangelogExportCmd = &cobra.Command{
	Use:     "export",
	Aliases: []string{"format"},
	Short:   "Formats the change log between two builds",
	Long: `Formats the change log between two builds.

For example:

    yontrack build changelog --from 1 --to 2

Additional options are available for the formatting:

	yontrack build changelog --from 1 --to 2 \
		--format markdown \
		--grouping "Bugs=bug|Features=features" \
		--alt-group "Misc" \
		--exclude delivery

The change log is available directly in the standard output.
	`,
	RunE: func(cmd *cobra.Command, args []string) error {

		// Boundaries

		from, err := cmd.Flags().GetInt("from")
		if err != nil {
			return err
		}

		fromPromotion, err := cmd.Flags().GetString("from-promotion")
		if err != nil {
			return err
		}

		to, err := cmd.Flags().GetInt("to")
		if err != nil {
			return err
		}

		// Format options

		format, err := cmd.Flags().GetString("format")
		if err != nil {
			return err
		}

		grouping, err := cmd.Flags().GetString("grouping")
		if err != nil {
			return err
		}

		altGroup, err := cmd.Flags().GetString("alt-group")
		if err != nil {
			return err
		}

		exclude, err := cmd.Flags().GetString("exclude")
		if err != nil {
			return err
		}

		// Computing the boundaries (to)
		var toId int
		if to != 0 {
			toId = to
		} else {
			toId, err = getToBoundary()
			if err != nil {
				return err
			}
		}

		// Computing the boundaries (from)
		fromId, err := getFromBoundary(toId, from, fromPromotion)
		if err != nil {
			return err
		}
		if fromId == 0 {
			return nil
		}

		// Query

		query := `
			query ExportChangeLog(
				$from: Int!,
				$to: Int!,
				$request: SCMChangeLogExportInput!
			) {
				scmChangeLog(from: $from, to: $to) {
					export(request: $request)
				}
			}
		`

		// Request

		request := make(map[string]interface{})

		if format != "" {
			request["format"] = format
		}
		if grouping != "" {
			request["grouping"] = grouping
		}
		if altGroup != "" {
			request["altGroup"] = altGroup
		}
		if exclude != "" {
			request["exclude"] = exclude
		}

		// Data

		var data struct {
			ScmChangeLog struct {
				Export string
			}
		}

		// Getting the configuration

		cfg, err := config.GetSelectedConfiguration()
		if err != nil {
			return err
		}

		// Call

		if err := client.GraphQLCall(cfg, query, map[string]interface{}{
			"from":    fromId,
			"to":      toId,
			"request": request,
		}, &data); err != nil {
			return err
		}

		// Print the exported change log
		_, _ = fmt.Fprintf(os.Stdout, strings.TrimSpace(data.ScmChangeLog.Export))

		// OK
		return nil
	},
}

func getToBoundary() (int, error) {
	return utils.GetBuildIdFromEnv()
}

func getFromBoundary(to int, from int, promotion string) (int, error) {
	if from > 0 {
		return from, nil
	} else if promotion != "" {
		return getFromPromotion(to, promotion)
	}
	return 0, fmt.Errorf("either --from or --from-promotion must be specified")
}

func getFromPromotion(to int, promotion string) (int, error) {

	// Getting the branch for the "to" boundary
	project, branch, err := getBranchForBuild(to)
	if err != nil {
		return 0, err
	}

	// Getting the last build having the promotion level
	query := `
	   query LastBuildWithPromotion($project: String!, $branch: String!, $promotion: String!) {
			builds(project: $project, branch: $branch, buildBranchFilter: {
				count: 1,
				withPromotionLevel: $promotion
			}) {
				id
			}
       }
    `

	variables := map[string]interface{}{
		"project":   project,
		"branch":    branch,
		"promotion": promotion,
	}

	var data struct {
		Builds []struct {
			Id string
		}
	}

	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return 0, err
	}

	if err := client.GraphQLCall(cfg, query, variables, &data); err != nil {
		return 0, err
	}

	if len(data.Builds) == 0 {
		return 0, nil
	} else {
		id, err := strconv.Atoi(data.Builds[0].Id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}
}

func getBranchForBuild(buildId int) (string, string, error) {
	query := `
		query BuildBranch($buildId: Int!) {
			build(id: $buildId) {
				branch {
					name
					project {
						name
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"buildId": buildId,
	}

	var data struct {
		Build struct {
			Branch struct {
				Name    string
				Project struct {
					Name string
				}
			}
		}
	}

	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return "", "", err
	}

	if err := client.GraphQLCall(cfg, query, variables, &data); err != nil {
		return "", "", err
	}

	return data.Build.Branch.Project.Name, data.Build.Branch.Name, nil
}

func init() {
	buildChangelogCmd.AddCommand(buildChangelogExportCmd)

	buildChangelogExportCmd.Flags().String("format", "", "Format of the changelog: text (default), markdown or html")
	buildChangelogExportCmd.Flags().String("grouping", "", "Grouping specification (see Yontrack doc)")
	buildChangelogExportCmd.Flags().String("alt-group", "", "Name of the group for unclassified issues")
	buildChangelogExportCmd.Flags().String("exclude", "", "Comma separated list of issue types to ignore")
}
