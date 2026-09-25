package cmd

import (
	"io"
	"os"
	"slices"
	client "yontrack/client"
	config "yontrack/config"
	"yontrack/utils"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type AutoPromotions struct {
	// List of validations and their configuration
	Validations []ValidationConfig
	// List of promotions
	Promotions []PromotionConfig
}

type ValidationConfig struct {
	// Name of the validation
	Name string
	// Optional description for the validation
	Description string
	// Optional data type
	DataType *string
	// Optional data type config
	DataTypeConfig *string
	// Test configuration
	Tests *TestSummaryValidationConfig
}

type TestSummaryValidationConfig struct {
	// Warning if skipped tests
	WarningIfSkipped bool
	// Failure if no tests
	FailWhenNoResults bool
}

type PromotionConfig struct {
	// Name of the promotion
	Name string
	// List of validations
	Validations []string
	// List of promotions
	Promotions []string
}

var promotionLevelAutoCmd = &cobra.Command{
	Use:   "auto",
	Short: "Sets up promotions and their auto promotions criteria using local YAML file",
	Long: `Sets up promotions and their auto promotions criteria using local YAML file.

	yontrack pl auto -p PROJECT -b BRANCH -l PROMOTION

By default, the definition of the promotions and their auto promotion is available in a local (current directory)
.ontrack/promotions.yaml file but this can be configured using the option:

    --yaml .ontrack/promotions.yaml

This YAML file has the following structure (example):

validations:
	- name: unit-tests
	  description: Unit tests
	  tests:
		warningIfSkipped: false
		failWhenNoResults: false
promotions:
	- name: BRONZE
	  validations:
		- unit-tests
		- lint
	- name: SILVER
	  promotions:
		- BRONZE
	  validations:
		- deploy
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		project, branch, err := utils.GetProjectBranchFlags(cmd, false, true)
		if err != nil {
			return err
		}

		promotionYamlPath, err := cmd.Flags().GetString("yaml")
		if err != nil {
			return err
		}
		if promotionYamlPath == "" {
			promotionYamlPath = ".ontrack/promotions.yaml"
		}

		// Configuration
		cfg, err := config.GetSelectedConfiguration()
		if err != nil {
			return err
		}

		// Reading the promotions.yaml file
		var root AutoPromotions
		reader, err := os.Open(promotionYamlPath)
		if err != nil {
			return err
		}
		buf, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(buf, &root)
		if err != nil {
			return err
		}

		// Setup of validations
		var createdValidations []string
		for _, validation := range root.Validations {
			createdValidations = append(createdValidations, validation.Name)
			// TODO Generic data type
			// Tests data type
			if validation.Tests != nil {
				err = SetupTestValidationStamp(
					project,
					branch,
					validation.Name,
					validation.Description,
					validation.Tests.WarningIfSkipped,
					validation.Tests.FailWhenNoResults,
				)
				if err != nil {
					return err
				}
			}
		}

		// List of validations and promotions to setup
		var validationStamps []string

		// Going over all promotions
		for _, promotion := range root.Promotions {
			if len(promotion.Validations) > 0 {
				validationStamps = append(validationStamps, promotion.Validations...)
			}
		}

		// Creates all the validations
		for _, validation := range validationStamps {
			// Check if not already created
			if slices.Index(createdValidations, validation) < 0 {
				// Setup the validation stamp
				err := client.SetupValidationStamp(
					cfg,
					project,
					branch,
					validation,
					"",
					"",
					nil,
				)
				if err != nil {
					return err
				}
			}
		}

		// Auto promotion setup
		for _, promotion := range root.Promotions {
			// Setup the promotion level
			err := client.SetupPromotionLevel(
				cfg,
				project,
				branch,
				promotion.Name,
				"",
				len(promotion.Validations) > 0 || len(promotion.Promotions) > 0,
				promotion.Validations,
				promotion.Promotions,
				"",
				"",
			)
			if err != nil {
				return err
			}
		}

		// OK
		return nil
	},
}

func init() {
	promotionLevelCmd.AddCommand(promotionLevelAutoCmd)
	promotionLevelAutoCmd.Flags().StringP("yaml", "y", ".ontrack/promotions.yaml", "Path to the YAML file")
}
