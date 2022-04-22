# route53copy, copies resource records between two AWS Route53 accounts

`route53copy` copies resource records from one AWS account to another. It
creates a `ChangeResourceRecordSet` with `UPSERT` for all `ResourceRecord`s of
the source account and sends it to the destination account.

The top-level `SOA` and `NS` are not included in the change set since they
should already exist in the destination account.

The domain must already exist in both accounts and [AWS Named Profiles](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html#cli-multiple-profiles)
must be configured for both the source account and the destination account.

It can also update the NS for a domain if you are using AWS as your registrar, and the domain is on the target account.

## Installation

Download the lastest version from [releases](https://github.com/pedrokiefer/route53copy/releases) page.

Or build it yourself by cloning this repository.

## Usage

```
$ route53copy --help
Route53Copy is a tool to copy records from one AWS account to another

Usage:
  route53copy <source_profile> <dest_profile> <domain> [flags]

Flags:
      --dry         Dry run
  -h, --help        help for route53copy
      --update-ns   Update nameserver records
  -v, --version     version for route53copy
```

```
$ route53copy aws_profile1 aws_profile2 example.com
Number of Records:  55
53 records in 'example.com' are copied from aws_profile1-dev to aws_profile2
{
  Comment: "Importing ALL records from aws_profile",
  Id: "/change/C3QI8LAP4H5G9",
  Status: "PENDING",
  SubmittedAt: 2015-09-25 08:47:19.908 +0000 UTC
}
```

## Release Notes

A list of changes are in the [RELEASE_NOTES](RELEASE_NOTES.md).

