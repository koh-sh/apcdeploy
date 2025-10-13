package main

import (
	_ "embed"

	"github.com/koh-sh/apcdeploy/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

//go:embed llms.md
var llmsContent string

func main() {
	cmd.SetVersionInfo(version, commit, date)
	cmd.SetLLMsContent(llmsContent)
	cmd.Execute()
}
