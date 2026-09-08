package client

import (
	"encoding/json"
	"errors"
	"fmt"
	resty "github.com/go-resty/resty/v2"
	"net/http"
	"time"
	config "yontrack/config"
)

// GraphQLCall performs a GraphQL query/mutation to Yontrack
func GraphQLCall(cfg *config.Config, query string, variables map[string]interface{}, data interface{}) error {

	// If config is disabled, skips the call
	if cfg.Disabled {
		return nil
	}

	body := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	client := resty.New()
	client.SetDebug(config.GraphQLLogging)
	if cfg.Token != "" {
		client.SetHeader("X-Ontrack-Token", cfg.Token)
	} else if cfg.Username != "" {
		client.SetBasicAuth(cfg.Username, cfg.Password)
	}

	if cfg.ConnectionRetry.MaxCount != 0 {
		client.SetRetryCount(cfg.ConnectionRetry.MaxCount).
			SetRetryMaxWaitTime(time.Duration(cfg.ConnectionRetry.MaxWaitTimeSec) * time.Second).
			AddRetryCondition(func(r *resty.Response, err error) bool {
				return err != nil || r.StatusCode() == http.StatusRequestTimeout || r.StatusCode() >= 500
			})
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post(cfg.URL + "/graphql")
	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("%s:\n%s", resp.Status(), resp.Body())
	}

	// Error returned
	var error struct {
		Status  int
		Message string
	}
	if err := json.Unmarshal(resp.Body(), &error); err == nil {
		if error.Status != 0 {
			return fmt.Errorf("HTTP %d %s", error.Status, error.Message)
		}
	}

	// Parsing
	result := &graphResponse{
		Data: data,
	}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return err
	}

	// Management of errors
	if result.Errors != nil {
		if len(result.Errors) > 0 {
			var message string
			for index, error := range result.Errors {
				message += fmt.Sprintf("%d) %s\n", index+1, error.Message)
			}
			return errors.New(message)
		}
	}

	// OK
	return nil
}

type graphErr struct {
	Message string
}

type graphResponse struct {
	Data   interface{}
	Errors []graphErr
}

// CheckDataErrors Given a list of errors in a data GraphQL structure (typically
// returned by a mutation), returns a GoLang error aggregating all error messages
// or returns nil if there is no error.
func CheckDataErrors(errorsList []struct{ Message string }) error {
	if errorsList != nil {
		if len(errorsList) > 0 {
			var message string
			for index, error := range errorsList {
				message += fmt.Sprintf("%d) %s\n", index+1, error.Message)
			}
			return errors.New(message)
		}
	}
	// All good
	return nil
}
