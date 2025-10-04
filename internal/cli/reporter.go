package cli

import (
	"context"
	"fmt"

	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/display"
	initPkg "github.com/koh-sh/apcdeploy/internal/init"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// Reporter implements the reporter.ProgressReporter interface for CLI output
type Reporter struct{}

// Ensure Reporter implements the interface
var _ reporter.ProgressReporter = (*Reporter)(nil)

// NewReporter creates a new CLI reporter
func NewReporter() *Reporter {
	return &Reporter{}
}

func (r *Reporter) Progress(message string) {
	fmt.Println(display.Progress(message))
}

func (r *Reporter) Success(message string) {
	fmt.Println(display.Success(message))
}

func (r *Reporter) Warning(message string) {
	fmt.Println(display.Warning(message))
}

// CreateInitializer creates a new initializer with AWS client and reporter
func CreateInitializer(ctx context.Context, region string) (*initPkg.Initializer, error) {
	awsClient, err := awsInternal.NewClient(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	reporter := NewReporter()
	return initPkg.New(awsClient, reporter), nil
}

// ShowInitNextSteps displays next steps after initialization
func ShowInitNextSteps(result *initPkg.Result) {
	fmt.Println("\n" + display.Success("Initialization complete!"))
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review the generated configuration files")
	fmt.Println("  2. Modify the data file as needed")
	fmt.Println("  3. Run 'apcdeploy diff' to preview changes")
	fmt.Println("  4. Run 'apcdeploy deploy' to deploy your configuration")

	// Suppress unused variable warning
	_ = result
}
