package lsresources

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// FormatJSON encodes the resources tree as indented JSON. When showStrategies
// is false, deployment strategies are omitted from the payload.
func FormatJSON(tree *ResourcesTree, showStrategies bool) ([]byte, error) {
	if !showStrategies {
		treeCopy := *tree
		treeCopy.DeploymentStrategies = nil
		tree = &treeCopy
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(tree); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderHumanReadable renders the resources tree through Reporter primitives
// (Header / Table). Color and ANSI handling stay inside the Reporter
// implementation; this function only describes structure and content.
//
// The deployment-strategies section is included only when showStrategies is
// true. When applications or strategies are absent, an explicit "no
// resources" message is emitted via Reporter.Info so silent mode stays quiet
// while still surfacing the empty state in normal mode.
func RenderHumanReadable(r reporter.Reporter, tree *ResourcesTree, showStrategies bool) {
	r.Header(fmt.Sprintf("Region: %s", tree.Region))

	if showStrategies {
		r.Header("Deployment Strategies")
		if len(tree.DeploymentStrategies) == 0 {
			r.Info("No deployment strategies found.")
		} else {
			rows := make([][]string, 0, len(tree.DeploymentStrategies))
			for _, s := range tree.DeploymentStrategies {
				rows = append(rows, []string{
					s.Name,
					s.ID,
					fmt.Sprintf("%dm", s.DeploymentDurationInMinutes),
					fmt.Sprintf("%dm", s.FinalBakeTimeInMinutes),
					fmt.Sprintf("%.1f%%", s.GrowthFactor),
					s.GrowthType,
					s.Description,
				})
			}
			r.Table(
				[]string{"Name", "ID", "Duration", "Bake Time", "Growth", "Type", "Description"},
				rows,
			)
		}
	}

	if len(tree.Applications) == 0 {
		r.Header("Applications")
		r.Info("No applications found.")
		return
	}

	for _, app := range tree.Applications {
		r.Header(fmt.Sprintf("Application: %s (ID: %s)", app.Name, app.ID))

		profileRows := make([][]string, 0, len(app.Profiles))
		for _, p := range app.Profiles {
			profileRows = append(profileRows, []string{p.Name, p.ID})
		}
		if len(profileRows) == 0 {
			r.Info("No configuration profiles.")
		} else {
			r.Table([]string{"Configuration Profile", "ID"}, profileRows)
		}

		envRows := make([][]string, 0, len(app.Environments))
		for _, e := range app.Environments {
			envRows = append(envRows, []string{e.Name, e.ID})
		}
		if len(envRows) == 0 {
			r.Info("No environments.")
		} else {
			r.Table([]string{"Environment", "ID"}, envRows)
		}
	}
}
