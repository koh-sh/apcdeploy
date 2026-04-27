package lsresources

import (
	"encoding/json"
	"strings"
	"testing"

	reporterTesting "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

func TestFormatJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		tree           *ResourcesTree
		showStrategies bool
		validate       func(*testing.T, string)
	}{
		{
			name: "full tree includes strategies when showStrategies is true",
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
			showStrategies: true,
			validate: func(t *testing.T, output string) {
				var result ResourcesTree
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("invalid JSON output: %v", err)
				}
				if result.Region != "us-east-1" {
					t.Errorf("expected region us-east-1, got %s", result.Region)
				}
				if len(result.Applications) != 1 {
					t.Errorf("expected 1 application, got %d", len(result.Applications))
				}
				if len(result.DeploymentStrategies) != 1 {
					t.Errorf("expected 1 deployment strategy, got %d", len(result.DeploymentStrategies))
				}
				if result.DeploymentStrategies[0].GrowthFactor != 100 {
					t.Errorf("expected growth factor 100, got %f", result.DeploymentStrategies[0].GrowthFactor)
				}
			},
		},
		{
			name: "showStrategies false omits deployment strategies",
			tree: &ResourcesTree{
				Region: "us-east-1",
				DeploymentStrategies: []DeploymentStrategy{
					{Name: "AppConfig.AllAtOnce", ID: "strategy-1"},
				},
				Applications: []Application{
					{Name: "app1", ID: "app-id-1"},
				},
			},
			showStrategies: false,
			validate: func(t *testing.T, output string) {
				var result ResourcesTree
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("invalid JSON output: %v", err)
				}
				if len(result.DeploymentStrategies) != 0 {
					t.Errorf("expected 0 strategies, got %d", len(result.DeploymentStrategies))
				}
				if len(result.Applications) != 1 {
					t.Errorf("expected 1 application, got %d", len(result.Applications))
				}
			},
		},
		{
			name: "empty tree marshals successfully",
			tree: &ResourcesTree{
				Region:               "us-west-2",
				Applications:         []Application{},
				DeploymentStrategies: []DeploymentStrategy{},
			},
			showStrategies: true,
			validate: func(t *testing.T, output string) {
				var result ResourcesTree
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("invalid JSON output: %v", err)
				}
				if len(result.Applications) != 0 {
					t.Errorf("expected 0 applications, got %d", len(result.Applications))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload, err := FormatJSON(tt.tree, tt.showStrategies)
			if err != nil {
				t.Fatalf("FormatJSON returned error: %v", err)
			}
			tt.validate(t, string(payload))
		})
	}
}

func TestRenderHumanReadable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		tree             *ResourcesTree
		showStrategies   bool
		expectedHeaders  []string
		expectedTableHas []string
		expectedInfo     []string
		denyHeaders      []string
	}{
		{
			name: "full tree with strategies and applications",
			tree: &ResourcesTree{
				Region: "us-east-1",
				DeploymentStrategies: []DeploymentStrategy{
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
						},
						Environments: []Environment{
							{Name: "dev", ID: "env-id-1"},
						},
					},
				},
			},
			showStrategies: true,
			expectedHeaders: []string{
				"Region: us-east-1",
				"Deployment Strategies",
				"Application: app1 (ID: app-id-1)",
			},
			expectedTableHas: []string{
				"AppConfig.Linear", "strategy-2", "Linear deployment", "30m", "10m", "20.0%", "LINEAR",
				"profile1", "prof-id-1",
				"dev", "env-id-1",
			},
		},
		{
			name: "showStrategies false skips strategies header",
			tree: &ResourcesTree{
				Region: "us-east-1",
				DeploymentStrategies: []DeploymentStrategy{
					{Name: "AppConfig.AllAtOnce", ID: "strategy-1"},
				},
				Applications: []Application{
					{
						Name:         "app1",
						ID:           "app-id-1",
						Profiles:     []ConfigurationProfile{{Name: "profile1", ID: "prof-id-1"}},
						Environments: []Environment{{Name: "dev", ID: "env-id-1"}},
					},
				},
			},
			showStrategies: false,
			expectedHeaders: []string{
				"Region: us-east-1",
				"Application: app1 (ID: app-id-1)",
			},
			expectedTableHas: []string{"profile1", "dev"},
			denyHeaders:      []string{"Deployment Strategies"},
		},
		{
			name: "no applications surfaces info message",
			tree: &ResourcesTree{
				Region:               "us-west-2",
				Applications:         []Application{},
				DeploymentStrategies: []DeploymentStrategy{},
			},
			showStrategies: true,
			expectedHeaders: []string{
				"Region: us-west-2",
				"Deployment Strategies",
				"Applications",
			},
			expectedInfo: []string{"No deployment strategies found.", "No applications found."},
		},
		{
			name: "application with no profiles or environments surfaces info messages",
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
			showStrategies: false,
			expectedHeaders: []string{
				"Application: empty-app (ID: app-empty)",
			},
			expectedInfo: []string{"No configuration profiles.", "No environments."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &reporterTesting.MockReporter{}
			RenderHumanReadable(r, tt.tree, tt.showStrategies)

			for _, h := range tt.expectedHeaders {
				if !r.HasMessage("header: " + h) {
					t.Errorf("expected Header(%q); messages=%v", h, r.Messages)
				}
			}
			for _, h := range tt.denyHeaders {
				if r.HasMessage("header: " + h) {
					t.Errorf("did not expect Header(%q); messages=%v", h, r.Messages)
				}
			}

			tableContent := flattenTables(r.Tables)
			for _, want := range tt.expectedTableHas {
				if !strings.Contains(tableContent, want) {
					t.Errorf("expected table cells to contain %q; got %q", want, tableContent)
				}
			}

			for _, info := range tt.expectedInfo {
				if !r.HasMessage("info: " + info) {
					t.Errorf("expected Info(%q); messages=%v", info, r.Messages)
				}
			}

			// Stdout must remain empty in human-readable mode (Data only used
			// for JSON output).
			if len(r.Stdout) != 0 {
				t.Errorf("expected empty stdout; got %q", r.Stdout)
			}
		})
	}
}

func flattenTables(tables []reporterTesting.TableCall) string {
	var b strings.Builder
	for _, tbl := range tables {
		for _, row := range tbl.Rows {
			for _, cell := range row {
				b.WriteString(cell)
				b.WriteByte('|')
			}
		}
	}
	return b.String()
}
