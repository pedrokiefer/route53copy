package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/pedrokiefer/route53copy/cmd"
	"github.com/pedrokiefer/route53copy/pkg/dns"
)

func main() {
	log.SetFlags(0)

	var version bool
	var dry bool
	var help bool
	var updateNS bool

	program := path.Base(os.Args[0])
	flag.BoolVar(&dry, "dry", false, "Don't make any changes")
	flag.BoolVar(&help, "help", false, "Show help text")
	flag.BoolVar(&version, "version", false, "Show version")
	flag.BoolVar(&updateNS, "update-ns", false, "Update NS records if domain is registered on destination account")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <source_profile> <dest_profile> <domain>\n", program)
		flag.PrintDefaults()
	}
	flag.Parse()

	if help {
		flag.Usage()
		os.Exit(0)
	}
	if version {
		fmt.Println(cmd.Version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "Wrong number of arguments, %d < 3\n", len(args))
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <source_profile> <dest_profile> <domain>\n", program)
		flag.PrintDefaults()
		os.Exit(1)
	}

	ctx := context.Background()
	sourceProfile := args[0]
	destProfile := args[1]
	domain := args[2]

	srcService := dns.NewRouteCopy(ctx, sourceProfile)
	dstService := dns.NewRouteCopy(ctx, destProfile)

	zone, err := srcService.GetHostedZone(ctx, domain)
	if err != nil {
		panic(err)
	}
	srcZoneID := aws.ToString(zone.Id)

	recordSets, err := srcService.GetResourceRecords(ctx, srcZoneID)
	if err != nil {
		panic(err)
	}

	changes := srcService.CreateChanges(domain, recordSets)
	log.Println("Number of records to copy", len(changes))
	if dry {
		log.Printf("Not copying records to %s since --dry is given\n", destProfile)
		zone, err := dstService.GetHostedZone(ctx, domain)
		if err != nil {
			panic(err)
		}

		log.Printf("Destination profile contains %d records, including NS and SOA\n",
			*zone.ResourceRecordSetCount)
	} else {
		zone, err := dstService.GetOrCreateZone(ctx, domain)
		if err != nil {
			panic(err)
		}
		dstZoneID := aws.ToString(zone.Id)

		changeInfo, err := dstService.UpdateRecords(ctx, sourceProfile, dstZoneID, changes)
		if err != nil {
			panic(err)
		}
		log.Printf("%d records in '%s' were copied from %s to %s\n",
			len(changes), domain, sourceProfile, destProfile)

		if changeInfo.Status != rtypes.ChangeStatusInsync {
			start := time.Now()
			err = dstService.WaitForChange(ctx, aws.ToString(changeInfo.Id), 2*time.Minute)
			if err != nil {
				panic(err)
			}
			log.Printf("%d records in '%s' are in sync after %s\n", len(changes), domain, time.Since(start))
		}

		if updateNS {
			log.Println("Updating NS records")
			updated, err := dstService.UpdateNSRecords(ctx, domain, dstZoneID)
			if err != nil {
				panic(err)
			}

			if updated {
				log.Printf("Registrar NS records for '%s' updated\n", domain)
			} else {
				log.Printf("Registrar NS records for '%s' are already up to date\n", domain)
			}
		}
	}
}
