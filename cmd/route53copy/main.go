package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/pedrokiefer/route53copy/cmd"
	"github.com/pedrokiefer/route53copy/pkg/dns"
)

func main() {
	log.SetFlags(0)

	var version bool
	var dry bool
	var help bool

	program := path.Base(os.Args[0])
	flag.BoolVar(&dry, "dry", false, "Don't make any changes")
	flag.BoolVar(&help, "help", false, "Show help text")
	flag.BoolVar(&version, "version", false, "Show version")
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
	recordSets, err := srcService.GetResourceRecords(ctx, domain)
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
		changeInfo, err := dstService.UpdateRecords(ctx, sourceProfile, domain, changes)
		if err != nil {
			panic(err)
		}
		log.Printf("%d records in '%s' are copied from %s to %s\n",
			len(changes), domain, sourceProfile, destProfile)
		log.Printf("%#v\n", changeInfo)
	}
}
