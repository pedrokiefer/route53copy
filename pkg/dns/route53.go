package dns

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type RouteCopy struct {
	cli *route53.Client
}

type HostedZoneNotFound struct {
	Zone string
}

func (e *HostedZoneNotFound) Error() string {
	return fmt.Sprintf("hosted zone not found: %s", e.Zone)
}

func NewRouteCopy(ctx context.Context, profile string) *RouteCopy {
	if r := os.Getenv("AWS_REGION"); r == "" {
		os.Setenv("AWS_REGION", "us-east-1")
	}
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

	fmt.Printf("Found %d hosted zones, trun %+v\n", len(resp.HostedZones), resp.IsTruncated)

	zone := resp.HostedZones[0]
	if *zone.Name != normalizeDomain(domain) {
		return rtypes.HostedZone{}, &HostedZoneNotFound{Zone: domain}
	}
	return zone, nil
}

func (r *RouteCopy) CreateZone(ctx context.Context, domain string) (rtypes.HostedZone, error) {
	params := &route53.CreateHostedZoneInput{
		Name:            aws.String(normalizeDomain(domain)),
		CallerReference: aws.String(fmt.Sprintf("%s-%d", domain, time.Now().Unix())),
		HostedZoneConfig: &rtypes.HostedZoneConfig{
			Comment:     aws.String("Created by route53copy"),
			PrivateZone: false,
		},
	}
	resp, err := r.cli.CreateHostedZone(ctx, params)
	if err != nil {
		return rtypes.HostedZone{}, err
	}
	return *resp.HostedZone, nil
}

func (r *RouteCopy) GetResourceRecords(ctx context.Context, domain string) ([]rtypes.ResourceRecordSet, error) {
	zone, err := r.GetHostedZone(ctx, domain)
	if err != nil {
		return nil, err
	}

	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(*zone.Id),
	}
	paginator := NewListResourceRecordSetsPaginator(r.cli, params)

	records := []rtypes.ResourceRecordSet{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return records, err
		}
		records = append(records, page.ResourceRecordSets...)
	}

	return records, nil
}

func (r *RouteCopy) CreateChanges(domain string, recordSets []rtypes.ResourceRecordSet) []rtypes.Change {
	domain = normalizeDomain(domain)
	var changes []rtypes.Change
	for _, recordSet := range recordSets {
		if (recordSet.Type == rtypes.RRTypeNs || recordSet.Type == rtypes.RRTypeSoa) && *recordSet.Name == domain {
			continue
		}
		change := rtypes.Change{
			Action: rtypes.ChangeActionUpsert,
			ResourceRecordSet: &rtypes.ResourceRecordSet{
				Name:                    recordSet.Name,
				Type:                    recordSet.Type,
				AliasTarget:             recordSet.AliasTarget,
				Failover:                recordSet.Failover,
				GeoLocation:             recordSet.GeoLocation,
				HealthCheckId:           recordSet.HealthCheckId,
				MultiValueAnswer:        recordSet.MultiValueAnswer,
				Region:                  recordSet.Region,
				ResourceRecords:         recordSet.ResourceRecords,
				SetIdentifier:           recordSet.SetIdentifier,
				TTL:                     recordSet.TTL,
				TrafficPolicyInstanceId: recordSet.TrafficPolicyInstanceId,
				Weight:                  recordSet.Weight,
			},
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
	var zone rtypes.HostedZone
	var err error
	zone, err = r.GetHostedZone(ctx, domain)
	if err != nil {
		var e *HostedZoneNotFound
		if errors.As(err, &e) {
			zone, err = r.CreateZone(ctx, domain)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: zone.Id,
		ChangeBatch: &rtypes.ChangeBatch{
			Changes: changes,
			Comment: aws.String("Importing ALL records from " + sourceProfile),
		},
	}
	resp, err := r.cli.ChangeResourceRecordSets(ctx, params)
	if err != nil {
		return nil, err
	}
	return resp.ChangeInfo, nil
}
