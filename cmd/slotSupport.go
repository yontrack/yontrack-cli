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
	"fmt"
	"yontrack/client"
	config "yontrack/config"
)

// slotBuild is the build carried by a pipeline.
type slotBuild struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// slotPipelineRef is the last pipeline to have been deployed on a slot, if any.
type slotPipelineRef struct {
	Id     string     `json:"id"`
	Number int        `json:"number"`
	Build  *slotBuild `json:"build"`
}

// slot is a project's deployment slot in an environment.
type slot struct {
	Id                   string           `json:"id"`
	LastDeployedPipeline *slotPipelineRef `json:"lastDeployedPipeline"`
}

// DeployedBuildId is the ID of the build the slot last deployed, or "" when the
// slot has never completed a deployment. Callers compare it against the build
// they are about to deploy, to decide whether there is anything to do.
func (s *slot) DeployedBuildId() string {
	if s.LastDeployedPipeline == nil || s.LastDeployedPipeline.Build == nil {
		return ""
	}
	return s.LastDeployedPipeline.Build.Id
}

// DeployedBuildName is the display name of the build the slot last deployed, or
// "" when the slot has never completed a deployment.
func (s *slot) DeployedBuildName() string {
	if s.LastDeployedPipeline == nil || s.LastDeployedPipeline.Build == nil {
		return ""
	}
	return s.LastDeployedPipeline.Build.DisplayName
}

// getSlot looks up the slot of a project in an environment, together with the
// build it last deployed.
func getSlot(cfg *config.Config, project string, environment string) (*slot, error) {
	var data struct {
		EnvironmentByName *struct {
			Slots []slot
		}
	}

	if err := client.GraphQLCall(cfg, `
		query GetSlot($environment: String!, $project: String!) {
			environmentByName(name: $environment) {
				slots(projects: [$project]) {
					id
					lastDeployedPipeline {
						id
						number
						build {
							id
							name
							displayName
						}
					}
				}
			}
		}
	`, map[string]interface{}{
		"environment": environment,
		"project":     project,
	}, &data); err != nil {
		return nil, err
	}

	if data.EnvironmentByName == nil {
		return nil, fmt.Errorf("no environment named %s", environment)
	}
	if len(data.EnvironmentByName.Slots) == 0 {
		return nil, fmt.Errorf("no slot for project %s in environment %s", project, environment)
	}

	found := data.EnvironmentByName.Slots[0]
	return &found, nil
}
