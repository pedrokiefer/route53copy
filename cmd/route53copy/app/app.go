package app

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/pedrokiefer/route53copy/pkg/dns"
	"github.com/spf13/cobra"
)

type App struct {
	SourceProfile      string
	DestinationProfile string
	Domain             string
	DryRun             bool
	UpdateNS           bool
}

func (a *App) Run(ctx context.Context) error {
	srcService := dns.NewRouteCopy(ctx, a.SourceProfile)
	dstService := dns.NewRouteCopy(ctx, a.DestinationProfile)

	zone, err := srcService.GetHostedZone(ctx, a.Domain)
	if err != nil {
		return err
	}
	srcZoneID := aws.ToString(zone.Id)

	recordSets, err := srcService.GetResourceRecords(ctx, srcZoneID)
	if err != nil {
		return err
	}

	changes := srcService.CreateChanges(a.Domain, recordSets)
	log.Println("Number of records to copy", len(changes))

	if a.DryRun {
		log.Printf("Not copying records to %s since --dry is given\n", a.DestinationProfile)
		zone, err := dstService.GetHostedZone(ctx, a.Domain)
		if err != nil {
			return err
		}

		log.Printf("Destination profile contains %d records, including NS and SOA\n",
			*zone.ResourceRecordSetCount)
	} else {
		zone, err := dstService.GetOrCreateZone(ctx, a.Domain)
		if err != nil {
			return err
		}
		dstZoneID := aws.ToString(zone.Id)

		changeInfo, err := dstService.UpdateRecords(ctx, a.SourceProfile, dstZoneID, changes)
		if err != nil {
			return err
		}
		log.Printf("%d records in '%s' were copied from %s to %s\n",
			len(changes), a.Domain, a.SourceProfile, a.DestinationProfile)

		if changeInfo.Status != rtypes.ChangeStatusInsync {
			start := time.Now()
			err = dstService.WaitForChange(ctx, aws.ToString(changeInfo.Id), 2*time.Minute)
			if err != nil {
				return err
			}
			log.Printf("%d records in '%s' are in sync after %s\n", len(changes), a.Domain, time.Since(start))
		}

		if a.UpdateNS {
			log.Println("Updating NS records")
			updated, err := dstService.UpdateNSRecords(ctx, a.Domain, dstZoneID)
			if err != nil {
				return err
			}

			if updated {
				log.Printf("Registrar NS records for '%s' updated\n", a.Domain)
			} else {
				log.Printf("Registrar NS records for '%s' are already up to date\n", a.Domain)
			}
		}
	}
	return nil
}

func NewCommand() *cobra.Command {
	a := App{}

	c := &cobra.Command{
		Use:   "route53copy <source_profile> <dest_profile> <domain>",
		Short: "Route53Copy is a tool to copy records from one AWS account to another",
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.SourceProfile = args[0]
			a.DestinationProfile = args[1]
			a.Domain = args[2]
			return a.Run(cmd.Context())
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	f := c.Flags()
	f.BoolVar(&a.DryRun, "dry", false, "Dry run")
	f.BoolVar(&a.UpdateNS, "update-ns", false, "Update nameserver records")
	return c
}
