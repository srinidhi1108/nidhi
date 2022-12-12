package providers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/civo/civogo"
	"github.com/digitalocean/godo"
	. "github.com/mlabouardy/komiser/models"
	"github.com/oracle/oci-go-sdk/common"
)

type FetchDataFunction func(ctx context.Context, client ProviderClient) ([]Resource, error)

type ProviderClient struct {
	AWSClient          *aws.Config
	DigitalOceanClient *godo.Client
	OciClient          common.ConfigurationProvider
	CivoClient         *civogo.Client
	Name               string
}
