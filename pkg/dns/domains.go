package dns

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53domains"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type DomainManager struct {
	cli    *route53domains.Client
	stscli *sts.Client
}

type Transfer struct {
	Password    string
	OperationID string
}

func NewDomainManager(ctx context.Context, profile string) (*DomainManager, error) {
	if r := os.Getenv("AWS_REGION"); r == "" {
		os.Setenv("AWS_REGION", "us-east-1")
	}
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profile))

	if err != nil {
		return nil, err
	}

	return &DomainManager{
		cli:    route53domains.NewFromConfig(cfg),
		stscli: sts.NewFromConfig(cfg),
	}, nil
}

func (dm *DomainManager) GetAccountID(ctx context.Context) (string, error) {
	i, err := dm.stscli.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}

	return aws.ToString(i.Account), nil
}

func (dm *DomainManager) ListRegisteredDomains(ctx context.Context) ([]string, error) {
	paginator := route53domains.NewListDomainsPaginator(dm.cli, &route53domains.ListDomainsInput{})

	domains := []string{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, domain := range page.Domains {
			domains = append(domains, aws.ToString(domain.DomainName))
		}
	}

	return domains, nil
}

func (dm *DomainManager) TransferDomain(ctx context.Context, domain, dstAccount string) (*Transfer, error) {
	resp, err := dm.cli.TransferDomainToAnotherAwsAccount(ctx, &route53domains.TransferDomainToAnotherAwsAccountInput{
		AccountId:  aws.String(dstAccount),
		DomainName: aws.String(domain),
	})
	if err != nil {
		return nil, err
	}

	return &Transfer{
		Password:    aws.ToString(resp.Password),
		OperationID: aws.ToString(resp.OperationId),
	}, nil
}

func (dm *DomainManager) CancelTranfer(ctx context.Context, domain string) (string, error) {
	resp, err := dm.cli.CancelDomainTransferToAnotherAwsAccount(ctx, &route53domains.CancelDomainTransferToAnotherAwsAccountInput{
		DomainName: aws.String(domain),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.OperationId), nil
}

func (dm *DomainManager) AcceptTransfer(ctx context.Context, domain, password string) (string, error) {
	resp, err := dm.cli.AcceptDomainTransferFromAnotherAwsAccount(ctx, &route53domains.AcceptDomainTransferFromAnotherAwsAccountInput{
		DomainName: aws.String(domain),
		Password:   aws.String(password),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.OperationId), nil
}
