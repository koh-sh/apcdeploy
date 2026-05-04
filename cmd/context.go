package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var llmsContent string

// SetLLMsContent sets the llms.md content from main
func SetLLMsContent(content string) {
	llmsContent = content
}

// ContextCommand creates and returns the context command
func ContextCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "context",
		Short: "Output context information for AI assistants",
		Long: `Output context information for AI assistants.
This command outputs the contents of llms.md, which provides guidelines
for AI assistants when using the apcdeploy command.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// CONTRACT EXCEPTION: context is a self-contained utility that
			// emits embedded llms.md to stdout with no Reporter primitives
			// involved (see CLAUDE.md "Command Structure" — context has no
			// internal/context package). Direct fmt.Print is intentional.
			fmt.Print(llmsContent)
			return nil
		},
	}
}
