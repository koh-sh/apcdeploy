package aws

import (
	"context"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/account"
	accountTypes "github.com/aws/aws-sdk-go-v2/service/account/types"
)

// ListEnabledRegions returns a sorted list of enabled regions for the account
func ListEnabledRegions(ctx context.Context, client AccountAPI) ([]string, error) {
	input := &account.ListRegionsInput{
		RegionOptStatusContains: []accountTypes.RegionOptStatus{
			accountTypes.RegionOptStatusEnabled,
			accountTypes.RegionOptStatusEnabledByDefault,
		},
	}

	output, err := client.ListRegions(ctx, input)
	if err != nil {
		return nil, err
	}

	regions := make([]string, 0, len(output.Regions))
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

	sort.Strings(regions)
	return regions, nil
}
