package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"yontrack/client"
	config "yontrack/config"
	"yontrack/utils"

	"github.com/spf13/cobra"
)

// buildSearchCmd represents the buildSearch command
var buildSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Searches for builds",
	Long: `Searches for builds.

Builds can be searched on a project using the '--project' flag:

    yontrack build search --project PROJECT

or on a branch using the '--branch' flag:

    yontrack build search --project PROJECT --branch BRANCH

In both cases, several criteria are available - see 'yontrack build search --help' to get their list. For example,
to look for a build using its commit:

    yontrack build search --project PROJECT --branch BRANCH --commit commit

By default, only the build names are printed, one per line.

You can change the display options using additional flags - see 'yontrack build search --help' to get their list.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		project, branch, err := utils.GetProjectBranchFlags(cmd, true, true)
		if err != nil {
			return err
		}

		// Project vs. branch search
		if branch == "" {
			return projectSearch(cmd, project)
		} else {
			return branchSearch(cmd, project, branch)
		}
	},
}

type build struct {
	Id          string
	Name        string
	DisplayName string
	Branch      struct {
		Id          string
		Name        string
		DisplayName string
		Project     struct {
			Id   string
			Name string
		}
	}
}

type buildList struct {
	Builds []build
}

func projectSearch(cmd *cobra.Command, project string) error {
	// Query
	query := `
		query BuildProjectSearch(
			$project: String!,
			$buildProjectFilter: BuildSearchForm!
		) {
			builds(
				project: $project,
				buildProjectFilter: $buildProjectFilter
			) {
				id
				name
				displayName
				branch {
					id
					name
					displayName
					project {
						id
						name
					}
				}
			}
		}
	`

	// Search form
	form := make(map[string]interface{})
	if err := rejectDisplayNameWithoutBranch(cmd); err != nil {
		return err
	}
	if err := fillFormWithProperty(cmd, &form, "property", "propertyValue"); err != nil {
		return err
	}
	if err := fillFormWithCount(cmd, &form, "maximumCount"); err != nil {
		return err
	}

	if err := fillFormWithWithPromotion(cmd, &form, "promotionName"); err != nil {
		return err
	}

	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	} else if name != "" {
		form["buildName"] = name
		nameExact, err := cmd.Flags().GetBool("name-exact")
		if err != nil {
			return err
		} else if nameExact {
			form["buildExactMatch"] = true
		}
	}

	// Gets the configuration
	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return err
	}

	// Result data
	var data buildList

	// Call
	if err := client.GraphQLCall(cfg, query, map[string]interface{}{
		"project":            project,
		"buildProjectFilter": form,
	}, &data); err != nil {
		return err
	}

	// Displaying the data
	return displayBuilds(cmd, &data)
}

func branchSearch(cmd *cobra.Command, project string, branch string) error {
	// Query
	query := `
		query BuildBranchSearch(
			$project: String!,
			$branch: String!,
			$buildBranchFilter: StandardBuildFilter!
		) {
			builds(
				project: $project,
				branch: $branch,
				buildBranchFilter: $buildBranchFilter
			) {
				id
				name
				displayName
				branch {
					id
					name
					displayName
					project {
						id
						name
					}
				}
			}
		}
	`

	// Search form
	form := make(map[string]interface{})
	if err := fillFormWithProperty(cmd, &form, "withProperty", "withPropertyValue"); err != nil {
		return err
	}
	if err := fillFormWithCount(cmd, &form, "count"); err != nil {
		return err
	}

	if err := fillFormWithWithPromotion(cmd, &form, "withPromotionLevel"); err != nil {
		return err
	}

	if err := fillFormWithDisplayName(cmd, &form, "withDisplayName"); err != nil {
		return err
	}

	// Gets the configuration
	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return err
	}

	// Result data
	var data buildList

	// Call
	if err := client.GraphQLCall(cfg, query, map[string]interface{}{
		"project":           project,
		"branch":            branch,
		"buildBranchFilter": form,
	}, &data); err != nil {
		return err
	}

	// Displaying the data
	return displayBuilds(cmd, &data)
}

func displayBuilds(cmd *cobra.Command, data *buildList) error {
	displayBranch, err := cmd.Flags().GetBool("display-branch")
	if err != nil {
		return err
	}
	displayId, err := cmd.Flags().GetBool("display-id")
	if err != nil {
		return err
	}
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	acceptNotFound, err := cmd.Flags().GetBool("accept-not-found")
	if err != nil {
		return err
	}

	if output == "" {
		for _, build := range data.Builds {
			if displayBranch {
				fmt.Printf(build.Branch.Name)
			} else if displayId {
				fmt.Println(build.Id)
			} else {
				fmt.Println(build.Name)
			}
		}
	} else {
		count, err := cmd.Flags().GetInt("count")
		if err != nil {
			return err
		}
		if count == 1 {
			if len(data.Builds) == 0 {
				if !acceptNotFound {
					return fmt.Errorf("no build found")
				}
			} else {
				build := data.Builds[0]
				return displayBuildOutput(build, output)
			}
		} else {
			return fmt.Errorf("output not supported with multiple builds. Set count to 1 or remove the output flag")
		}
	}

	return nil
}

func displayBuildOutput(build build, output string) error {
	if output == "env" {
		return displayBuildEnv(build)
	} else if output == "json" {
		return displayBuildJson(build)
	} else {
		return fmt.Errorf("unknown output format: %s", output)
	}
}

func displayBuildEnv(build build) error {
	_, _ = fmt.Fprintf(os.Stdout, "export YONTRACK_PROJECT_ID=%s\n", build.Branch.Project.Id)
	_, _ = fmt.Fprintf(os.Stdout, "export YONTRACK_PROJECT_NAME=%s\n", build.Branch.Project.Name)
	_, _ = fmt.Fprintf(os.Stdout, "export YONTRACK_BRANCH_ID=%s\n", build.Branch.Id)
	_, _ = fmt.Fprintf(os.Stdout, "export YONTRACK_BRANCH_NAME=%s\n", build.Branch.Name)
	_, _ = fmt.Fprintf(os.Stdout, "export YONTRACK_BUILD_ID=%s\n", build.Id)
	_, _ = fmt.Fprintf(os.Stdout, "export YONTRACK_BUILD_NAME=%s\n", build.Name)
	return nil
}

func displayBuildJson(build build) error {
	jsonBytes, err := json.MarshalIndent(build, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal build to JSON: %w", err)
	}
	_, err = fmt.Fprintf(os.Stdout, "%s\n", jsonBytes)
	return err
}

func fillFormWithCount(cmd *cobra.Command, form *map[string]interface{}, countField string) error {
	count, err := cmd.Flags().GetInt("count")
	if err != nil {
		return err
	}

	if count > 0 {
		(*form)[countField] = count
	}

	return nil
}

func fillFormWithProperty(cmd *cobra.Command, form *map[string]interface{}, propertyTypeField string, propertyValueField string) error {
	commit, err := cmd.Flags().GetString("commit")
	if err != nil {
		return err
	}
	property, err := cmd.Flags().GetString("with-property")
	if err != nil {
		return err
	}
	propertyValue, err := cmd.Flags().GetString("with-property-value")
	if err != nil {
		return err
	}

	propertyType, value, err := propertyCriteria(commit, property, propertyValue)
	if err != nil {
		return err
	}

	if propertyType != "" {
		(*form)[propertyTypeField] = propertyType
		// A property type on its own matches any build carrying it, whatever its
		// value, so an empty value is left out rather than sent as "".
		if value != "" {
			(*form)[propertyValueField] = value
		}
	}

	return nil
}

// propertyCriteria resolves which property to filter on, and on which value.
//
// The generic --with-property/--with-property-value pair takes any property
// type. --commit stays as the shorthand it has always been, and is simply a
// preset over the same mechanism.
//
// Returns empty strings when no property criterion was asked for.
func propertyCriteria(commit, property, propertyValue string) (string, string, error) {
	if commit != "" && property != "" {
		return "", "", errors.New("--commit and --with-property are mutually exclusive")
	}
	if property == "" && propertyValue != "" {
		return "", "", errors.New("--with-property-value requires --with-property")
	}

	if property != "" {
		return property, propertyValue, nil
	}
	if commit != "" {
		return gitCommitPropertyType, commit, nil
	}
	return "", "", nil
}

func fillFormWithWithPromotion(cmd *cobra.Command, form *map[string]interface{}, fieldName string) error {
	return fillForm(cmd, form, "with-promotion", fieldName)
}

// fillFormWithDisplayName adds the display-name criterion to a StandardBuildFilter.
//
// A build's display name is its release property when it has one, and its own
// name otherwise, so this is what matches the version a human would quote.
func fillFormWithDisplayName(cmd *cobra.Command, form *map[string]interface{}, fieldName string) error {
	return fillForm(cmd, form, "with-display-name", fieldName)
}

// rejectDisplayNameWithoutBranch fails a project-wide search that asked for a
// display name. Only StandardBuildFilter carries `withDisplayName`;
// BuildSearchForm, used when no branch is given, has no equivalent - so the
// criterion would otherwise be dropped and the search would quietly return the
// wrong builds.
func rejectDisplayNameWithoutBranch(cmd *cobra.Command) error {
	displayName, err := cmd.Flags().GetString("with-display-name")
	if err != nil {
		return err
	}
	if displayName != "" {
		return errors.New("--with-display-name requires --branch")
	}
	return nil
}

func fillForm(cmd *cobra.Command, form *map[string]interface{}, argName string, fieldName string) error {
	value, err := cmd.Flags().GetString(argName)
	if err != nil {
		return err
	}

	if value != "" {
		(*form)[fieldName] = value
	}

	return nil
}

func init() {
	buildCmd.AddCommand(buildSearchCmd)

	buildSearchCmd.Flags().StringP("project", "p", "", "Name of the project")
	buildSearchCmd.Flags().StringP("branch", "b", "", "Name of the branch")

	// Criteria
	buildSearchCmd.Flags().Int("count", 10, "Number of builds to return")
	buildSearchCmd.Flags().String("with-promotion", "", "Builds must have this promotion")
	buildSearchCmd.Flags().String("with-display-name", "", "Builds must have this display name (their release property, or their name when they have none). Requires --branch.")
	buildSearchCmd.Flags().String("name", "", "Builds must have this name or match this regular expression")
	buildSearchCmd.Flags().Bool("name-exact", true, "If present together with the `name` flag, requires an exact match.")

	// Property criteria
	buildSearchCmd.Flags().String("commit", "", "Commit for the build. Shorthand for --with-property "+gitCommitPropertyType+" --with-property-value <commit>.")
	buildSearchCmd.Flags().String("with-property", "", "Builds must carry this property, given as its fully qualified type name")
	buildSearchCmd.Flags().String("with-property-value", "", "Builds must carry --with-property with this value. Without it, any value matches.")

	// Display options
	buildSearchCmd.Flags().StringP("output", "o", "", "How to output the search results (env, json). Incompatible with the `display` options.")
	buildSearchCmd.Flags().Bool("display-id", false, "Displays the build ID instead of its name.")
	buildSearchCmd.Flags().Bool("display-branch", false, "Displays the build branch name instead of its name.")
	buildSearchCmd.Flags().BoolP("accept-not-found", "n", false, "If the search does not return any build, does not fail the command.")
}
