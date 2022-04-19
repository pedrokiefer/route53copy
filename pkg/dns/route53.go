package dns

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type RouteCopy struct {
	cli *route53.Client
}

func NewRouteCopy(ctx context.Context, profile string) *RouteCopy {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profile))
	if err != nil {
		panic(err)
	}
	return &RouteCopy{
		cli: route53.NewFromConfig(cfg),
	}
}

func (r *RouteCopy) GetHostedZone(ctx context.Context, domain string) (rtypes.HostedZone, error) {
	params := &route53.ListHostedZonesByNameInput{
		DNSName:  aws.String(domain),
		MaxItems: aws.Int32(1),
	}
	resp, err := r.cli.ListHostedZonesByName(ctx, params)
	if err != nil {
		return rtypes.HostedZone{}, err
	}

	zone := resp.HostedZones[0]
	return zone, nil
}

func (r *RouteCopy) GetResourceRecords(ctx context.Context, domain string) ([]rtypes.ResourceRecordSet, error) {
	zone, err := r.GetHostedZone(ctx, domain)
	if err != nil {
		return nil, err
	}

	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(*zone.Id),
	}
	resp, err := r.cli.ListResourceRecordSets(ctx, params)
	if err != nil {
		return nil, err
	}
	return resp.ResourceRecordSets, nil
}

func (r *RouteCopy) CreateChanges(domain string, recordSets []rtypes.ResourceRecordSet) []rtypes.Change {
	domain = normalizeDomain(domain)
	var changes []rtypes.Change
	for _, recordSet := range recordSets {
		if (recordSet.Type == rtypes.RRTypeNs || recordSet.Type == rtypes.RRTypeSoa) && *recordSet.Name == domain {
			continue
		}
		change := rtypes.Change{
			Action:            rtypes.ChangeActionUpsert,
			ResourceRecordSet: &recordSet,
		}
		changes = append(changes, change)
	}
	return changes

}

func normalizeDomain(domain string) string {
	if strings.HasSuffix(domain, ".") {
		return domain
	} else {
		return domain + "."
	}
}

func (r *RouteCopy) UpdateRecords(ctx context.Context, sourceProfile, domain string, changes []rtypes.Change) (*rtypes.ChangeInfo, error) {
	zone, err := r.GetHostedZone(ctx, domain)
	if err != nil {
		return nil, err
	}
	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: zone.Id,
		ChangeBatch: &rtypes.ChangeBatch{
			Changes: changes,
			Comment: aws.String("Importing ALL records from " + sourceProfile),
		},
	}
	resp, err := r.cli.ChangeResourceRecordSets(ctx, params)
	return resp.ChangeInfo, nil
}
