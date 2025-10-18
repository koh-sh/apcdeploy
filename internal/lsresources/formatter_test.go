package lsresources

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tree     *ResourcesTree
		validate func(*testing.T, string)
	}{
		{
			name: "full tree with multiple applications",
			tree: &ResourcesTree{
				Region: "us-east-1",
				DeploymentStrategies: []DeploymentStrategy{
					{
						Name:                        "AppConfig.AllAtOnce",
						ID:                          "strategy-1",
						Description:                 "Quick deployment",
						DeploymentDurationInMinutes: 0,
						FinalBakeTimeInMinutes:      0,
						GrowthFactor:                100,
						GrowthType:                  "LINEAR",
					},
				},
				Applications: []Application{
					{
						Name: "app1",
						ID:   "app-id-1",
						Profiles: []ConfigurationProfile{
							{Name: "profile1", ID: "prof-id-1"},
						},
						Environments: []Environment{
							{Name: "dev", ID: "env-id-1"},
						},
					},
				},
			},
			validate: func(t *testing.T, output string) {
				// Verify it's valid JSON
				var result ResourcesTree
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("invalid JSON output: %v", err)
				}

				// Verify structure
				if result.Region != "us-east-1" {
					t.Errorf("expected region us-east-1, got %s", result.Region)
				}
				if len(result.Applications) != 1 {
					t.Errorf("expected 1 application, got %d", len(result.Applications))
				}
				if len(result.DeploymentStrategies) != 1 {
					t.Errorf("expected 1 deployment strategy, got %d", len(result.DeploymentStrategies))
				}
				// Verify deployment strategy details
				if result.DeploymentStrategies[0].GrowthFactor != 100 {
					t.Errorf("expected growth factor 100, got %f", result.DeploymentStrategies[0].GrowthFactor)
				}
			},
		},
		{
			name: "empty applications",
			tree: &ResourcesTree{
				Region:               "us-west-2",
				Applications:         []Application{},
				DeploymentStrategies: []DeploymentStrategy{},
			},
			validate: func(t *testing.T, output string) {
				var result ResourcesTree
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("invalid JSON output: %v", err)
				}
				if len(result.Applications) != 0 {
					t.Errorf("expected 0 applications, got %d", len(result.Applications))
				}
				if len(result.DeploymentStrategies) != 0 {
					t.Errorf("expected 0 deployment strategies, got %d", len(result.DeploymentStrategies))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := FormatJSON(tt.tree, &buf, true)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			tt.validate(t, buf.String())
		})
	}
}

func TestFormatJSON_WithoutStrategies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tree     *ResourcesTree
		validate func(*testing.T, string)
	}{
		{
			name: "showStrategies false excludes deployment strategies",
			tree: &ResourcesTree{
				Region: "us-east-1",
				DeploymentStrategies: []DeploymentStrategy{
					{
						Name:                        "AppConfig.AllAtOnce",
						ID:                          "strategy-1",
						Description:                 "Quick deployment",
						DeploymentDurationInMinutes: 0,
						FinalBakeTimeInMinutes:      0,
						GrowthFactor:                100,
						GrowthType:                  "LINEAR",
					},
				},
				Applications: []Application{
					{
						Name: "app1",
						ID:   "app-id-1",
						Profiles: []ConfigurationProfile{
							{Name: "profile1", ID: "prof-id-1"},
						},
						Environments: []Environment{
							{Name: "dev", ID: "env-id-1"},
						},
					},
				},
			},
			validate: func(t *testing.T, output string) {
				var result ResourcesTree
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("invalid JSON output: %v", err)
				}

				// Verify deployment strategies are excluded
				if len(result.DeploymentStrategies) != 0 {
					t.Errorf("expected 0 deployment strategies with showStrategies=false, got %d", len(result.DeploymentStrategies))
				}

				// Verify applications are still included
				if len(result.Applications) != 1 {
					t.Errorf("expected 1 application, got %d", len(result.Applications))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := FormatJSON(tt.tree, &buf, false) // showStrategies = false
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			tt.validate(t, buf.String())
		})
	}
}

func TestFormatHumanReadable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		tree             *ResourcesTree
		expectedContains []string
	}{
		{
			name: "full tree with multiple applications",
			tree: &ResourcesTree{
				Region: "us-east-1",
				DeploymentStrategies: []DeploymentStrategy{
					{
						Name:                        "AppConfig.AllAtOnce",
						ID:                          "strategy-1",
						Description:                 "Quick deployment",
						DeploymentDurationInMinutes: 0,
						FinalBakeTimeInMinutes:      0,
						GrowthFactor:                100,
						GrowthType:                  "LINEAR",
					},
					{
						Name:                        "AppConfig.Linear",
						ID:                          "strategy-2",
						Description:                 "Linear deployment",
						DeploymentDurationInMinutes: 30,
						FinalBakeTimeInMinutes:      10,
						GrowthFactor:                20,
						GrowthType:                  "LINEAR",
					},
				},
				Applications: []Application{
					{
						Name: "app1",
						ID:   "app-id-1",
						Profiles: []ConfigurationProfile{
							{Name: "profile1", ID: "prof-id-1"},
							{Name: "profile2", ID: "prof-id-2"},
						},
						Environments: []Environment{
							{Name: "dev", ID: "env-id-1"},
							{Name: "prod", ID: "env-id-2"},
						},
					},
					{
						Name: "app2",
						ID:   "app-id-2",
						Profiles: []ConfigurationProfile{
							{Name: "profile3", ID: "prof-id-3"},
						},
						Environments: []Environment{
							{Name: "staging", ID: "env-id-3"},
						},
					},
				},
			},
			expectedContains: []string{
				"Region: us-east-1",
				"Deployment Strategies:",
				"AppConfig.AllAtOnce",
				"AppConfig.Linear",
				"Quick deployment",
				"Linear deployment",
				"Deployment Duration: 30 minutes",
				"Final Bake Time: 10 minutes",
				"Growth Factor: 20.0%",
				"Growth Factor: 100.0%",
				"Growth Type: LINEAR",
				"Applications:",
				"app1",
				"app2",
				"Configuration Profiles:",
				"profile1",
				"profile2",
				"profile3",
				"Environments:",
				"dev",
				"prod",
				"staging",
			},
		},
		{
			name: "empty applications",
			tree: &ResourcesTree{
				Region:               "us-west-2",
				Applications:         []Application{},
				DeploymentStrategies: []DeploymentStrategy{},
			},
			expectedContains: []string{
				"Region: us-west-2",
				"Deployment Strategies:",
				"No deployment strategies found",
				"No applications found",
			},
		},
		{
			name: "application with no profiles or environments",
			tree: &ResourcesTree{
				Region:               "eu-west-1",
				DeploymentStrategies: []DeploymentStrategy{},
				Applications: []Application{
					{
						Name:         "empty-app",
						ID:           "app-empty",
						Profiles:     []ConfigurationProfile{},
						Environments: []Environment{},
					},
				},
			},
			expectedContains: []string{
				"Region: eu-west-1",
				"Deployment Strategies:",
				"empty-app",
				"No configuration profiles",
				"No environments",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := FormatHumanReadable(tt.tree, &buf, true)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			output := buf.String()
			for _, expected := range tt.expectedContains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestFormatHumanReadable_WithoutStrategies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		tree             *ResourcesTree
		expectedContains []string
		notExpected      []string
	}{
		{
			name: "showStrategies false excludes deployment strategies section",
			tree: &ResourcesTree{
				Region: "us-east-1",
				DeploymentStrategies: []DeploymentStrategy{
					{
						Name:                        "AppConfig.AllAtOnce",
						ID:                          "strategy-1",
						Description:                 "Quick deployment",
						DeploymentDurationInMinutes: 0,
						FinalBakeTimeInMinutes:      0,
						GrowthFactor:                100,
						GrowthType:                  "LINEAR",
					},
				},
				Applications: []Application{
					{
						Name: "app1",
						ID:   "app-id-1",
						Profiles: []ConfigurationProfile{
							{Name: "profile1", ID: "prof-id-1"},
						},
						Environments: []Environment{
							{Name: "dev", ID: "env-id-1"},
						},
					},
				},
			},
			expectedContains: []string{
				"Region: us-east-1",
				"Applications:",
				"app1",
				"Configuration Profiles:",
				"profile1",
				"Environments:",
				"dev",
			},
			notExpected: []string{
				"Deployment Strategies:",
				"AppConfig.AllAtOnce",
				"Quick deployment",
				"Growth Factor:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := FormatHumanReadable(tt.tree, &buf, false) // showStrategies = false
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			output := buf.String()

			// Check expected content is present
			for _, expected := range tt.expectedContains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, output)
				}
			}

			// Check deployment strategies content is NOT present
			for _, notExpected := range tt.notExpected {
				if strings.Contains(output, notExpected) {
					t.Errorf("expected output NOT to contain %q, got:\n%s", notExpected, output)
				}
			}
		})
	}
}
