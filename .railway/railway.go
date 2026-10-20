package main

import (
	"os"

	"github.com/railwayapp/railway-go-sdk"
)

// Railway defines the existing production service. Application source arrives
// only through the published-release archive upload, never a GitHub branch.
func Railway(ctx railway.Context) map[string]any {
	projectID := os.Getenv("RAILWAY_PROJECT_ID")
	environmentID := os.Getenv("RAILWAY_ENVIRONMENT_ID")
	if projectID == "" || environmentID == "" ||
		ctx.ProjectID != projectID || ctx.EnvironmentID != environmentID ||
		ctx.Environment != "production" {
		panic("Railway configuration requires the configured production target")
	}

	app := railway.ServiceNamed("myjournal-app", railway.ServiceConfig{
		"build": map[string]any{
			"builder":          "DOCKERFILE",
			"dockerfilePath":   "Dockerfile",
			"buildEnvironment": "V3",
		},
		"replicas": map[string]any{"asia-southeast1-eqsg3a": 1},
		"deploy": map[string]any{
			"healthcheckPath":         "/healthz",
			"healthcheckTimeout":      300,
			"restartPolicyMaxRetries": 3,
			"runtime":                 "V2",
			"useLegacyStacker":        false,
			"ipv6EgressEnabled":       false,
		},
	})
	graph := railway.ProjectNamed("myjournal", []any{app}).Graph()
	// SDK v0.3.0 has no project variable-policy helper. CLI 5.68.0 accepts
	// this graph field and leaves every Railway variable outside IaC ownership.
	graph["variables"] = map[string]any{"managed": false}
	return graph
}
