package dns

import (
	"fmt"
	"log"
	"net"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	rdtypes "github.com/aws/aws-sdk-go-v2/service/route53domains/types"
	"github.com/miekg/dns"
)

var nsre = regexp.MustCompile(`.*NS.(.*)`)

type NSRecordNotFound struct {
	Domain string
}

func (e *NSRecordNotFound) Error() string {
	return fmt.Sprintf("failed to get nameservers for: %s", e.Domain)
}

func GetNameserversFor(domain string) ([]rdtypes.Nameserver, error) {
	config, _ := dns.ClientConfigFromFile("/etc/resolv.conf")

	c := &dns.Client{}

	m := &dns.Msg{}
	m.SetQuestion(dns.Fqdn(domain), dns.TypeNS)
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, net.JoinHostPort(config.Servers[0], config.Port))
	if err != nil {
		return nil, err
	}
	if r.Rcode != dns.RcodeSuccess {
		return nil, &NSRecordNotFound{Domain: domain}
	}

	nss := []rdtypes.Nameserver{}
	for _, rr := range r.Answer {
		if ns, ok := rr.(*dns.NS); ok {
			nsStr := ns.String()
			server := nsre.FindStringSubmatch(nsStr)[1]
			server = denormalizeDomain(server)
			log.Printf("[DEBUG] Found nameserver: %s", server)
			nss = append(nss, rdtypes.Nameserver{
				Name: aws.String(server),
			})
		}
	}
	return nss, nil
}
