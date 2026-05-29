package aws

import (
	"context"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/account"
	accountTypes "github.com/aws/aws-sdk-go-v2/service/account/types"
)

// ListEnabledRegions returns a sorted list of enabled regions for the account.
// It paginates through all pages of account.ListRegions to avoid silently truncating
// accounts whose enabled-region list spans multiple pages.
func ListEnabledRegions(ctx context.Context, client AccountAPI) ([]string, error) {
	var regions []string
	var nextToken *string

	for {
		output, err := client.ListRegions(ctx, &account.ListRegionsInput{
			RegionOptStatusContains: []accountTypes.RegionOptStatus{
				accountTypes.RegionOptStatusEnabled,
				accountTypes.RegionOptStatusEnabledByDefault,
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}

		for _, region := range output.Regions {
			// Skip regions with nil names
			if region.RegionName == nil {
				continue
			}
			// Defensive check: only include enabled or enabled by default regions
			// (API should filter, but this is a safety measure)
			if region.RegionOptStatus == accountTypes.RegionOptStatusEnabled ||
				region.RegionOptStatus == accountTypes.RegionOptStatusEnabledByDefault {
				regions = append(regions, aws.ToString(region.RegionName))
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	slices.Sort(regions)
	return regions, nil
}
