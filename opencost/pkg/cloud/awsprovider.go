package cloud

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/opencost/opencost/pkg/kubecost"

	"github.com/opencost/opencost/pkg/clustercache"
	"github.com/opencost/opencost/pkg/env"
	errs "github.com/opencost/opencost/pkg/errors"
	"github.com/opencost/opencost/pkg/log"
	"github.com/opencost/opencost/pkg/util"
	"github.com/opencost/opencost/pkg/util/fileutil"
	"github.com/opencost/opencost/pkg/util/json"

	awsSDK "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	athenaTypes "github.com/aws/aws-sdk-go-v2/service/athena/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/jszwec/csvutil"

	v1 "k8s.io/api/core/v1"
)

const (
	supportedSpotFeedVersion = "1"
	SpotInfoUpdateType       = "spotinfo"
	AthenaInfoUpdateType     = "athenainfo"
	PreemptibleType          = "preemptible"

	APIPricingSource              = "Public API"
	SpotPricingSource             = "Spot Data Feed"
	ReservedInstancePricingSource = "Savings Plan, Reserved Instance, and Out-Of-Cluster"
)

var (
	// It's of the form aws:///us-east-2a/i-0fea4fd46592d050b and we want i-0fea4fd46592d050b, if it exists
	provIdRx      = regexp.MustCompile("aws:///([^/]+)/([^/]+)")
	usageTypeRegx = regexp.MustCompile(".*(-|^)(EBS.+)")
	versionRx     = regexp.MustCompile("^#Version: (\\d+)\\.\\d+$")
)

func (aws *AWS) PricingSourceStatus() map[string]*PricingSource {

	sources := make(map[string]*PricingSource)

	sps := &PricingSource{
		Name:    SpotPricingSource,
		Enabled: true,
	}

	if !aws.SpotRefreshEnabled() {
		sps.Available = false
		sps.Error = "Spot instances not set up"
		sps.Enabled = false
	} else {
		sps.Error = ""
		if aws.SpotPricingError != nil {
			sps.Error = aws.SpotPricingError.Error()
		}
		if sps.Error != "" {
			sps.Available = false
		} else if len(aws.SpotPricingByInstanceID) > 0 {
			sps.Available = true
		} else {
			sps.Error = "No spot instances detected"
		}
	}
	sources[SpotPricingSource] = sps

	rps := &PricingSource{
		Name:    ReservedInstancePricingSource,
		Enabled: true,
	}
	rps.Error = ""
	if aws.RIPricingError != nil {
		rps.Error = aws.RIPricingError.Error()
	}
	if rps.Error != "" {
		rps.Available = false
	} else {
		rps.Available = true
	}
	sources[ReservedInstancePricingSource] = rps
	return sources

}

// How often spot data is refreshed
const SpotRefreshDuration = 15 * time.Minute

var awsRegions = []string{
	"us-east-2",
	"us-east-1",
	"us-west-1",
	"us-west-2",
	"ap-east-1",
	"ap-south-1",
	"ap-northeast-3",
	"ap-northeast-2",
	"ap-southeast-1",
	"ap-southeast-2",
	"ap-northeast-1",
	"ap-southeast-3",
	"ca-central-1",
	"cn-north-1",
	"cn-northwest-1",
	"eu-central-1",
	"eu-west-1",
	"eu-west-2",
	"eu-west-3",
	"eu-north-1",
	"eu-south-1",
	"me-south-1",
	"sa-east-1",
	"af-south-1",
	"us-gov-east-1",
	"us-gov-west-1",
}

// AWS represents an Amazon Provider
type AWS struct {
	Pricing                     map[string]*AWSProductTerms
	SpotPricingByInstanceID     map[string]*spotInfo
	SpotPricingUpdatedAt        *time.Time
	SpotRefreshRunning          bool
	SpotPricingLock             sync.RWMutex
	SpotPricingError            error
	RIPricingByInstanceID       map[string]*RIData
	RIPricingError              error
	RIDataRunning               bool
	RIDataLock                  sync.RWMutex
	SavingsPlanDataByInstanceID map[string]*SavingsPlanData
	SavingsPlanDataRunning      bool
	SavingsPlanDataLock         sync.RWMutex
	ValidPricingKeys            map[string]bool
	Clientset                   clustercache.ClusterCache
	BaseCPUPrice                string
	BaseRAMPrice                string
	BaseGPUPrice                string
	BaseSpotCPUPrice            string
	BaseSpotRAMPrice            string
	BaseSpotGPUPrice            string
	SpotLabelName               string
	SpotLabelValue              string
	SpotDataRegion              string
	SpotDataBucket              string
	SpotDataPrefix              string
	ProjectID                   string
	DownloadPricingDataLock     sync.RWMutex
	Config                      *ProviderConfig
	serviceAccountChecks        *ServiceAccountChecks
	clusterManagementPrice      float64
	clusterAccountId            string
	clusterRegion               string
	clusterProvisioner          string
	*CustomProvider
}

// AWSAccessKey holds AWS credentials and fulfils the awsV2.CredentialsProvider interface
type AWSAccessKey struct {
	AccessKeyID     string `json:"aws_access_key_id"`
	SecretAccessKey string `json:"aws_secret_access_key"`
}

// Retrieve returns a set of awsV2 credentials using the AWSAccessKey's key and secret.
// This fulfils the awsV2.CredentialsProvider interface contract.
func (accessKey AWSAccessKey) Retrieve(ctx context.Context) (awsSDK.Credentials, error) {
	return awsSDK.Credentials{
		AccessKeyID:     accessKey.AccessKeyID,
		SecretAccessKey: accessKey.SecretAccessKey,
	}, nil
}

// CreateConfig creates an AWS SDK V2 Config for the credentials that it contains for the provided region
func (accessKey AWSAccessKey) CreateConfig(region string) (awsSDK.Config, error) {
	var cfg awsSDK.Config
	var err error
	// If accessKey values have not been provided, attempt to load cfg from service key annotations
	if accessKey.AccessKeyID == "" && accessKey.SecretAccessKey == "" {
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
		if err != nil {
			return cfg, fmt.Errorf("failed to initialize AWS SDK config for region from annotation %s: %s", region, err)
		}
	} else {
		// The AWS SDK v2 requires an object fulfilling the CredentialsProvider interface, which cloud.AWSAccessKey does
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(accessKey), config.WithRegion(region))
		if err != nil {
			return cfg, fmt.Errorf("failed to initialize AWS SDK config for region %s: %s", region, err)
		}
	}

	return cfg, nil
}

// AWSPricing maps a k8s node to an AWS Pricing "product"
type AWSPricing struct {
	Products map[string]*AWSProduct `json:"products"`
	Terms    AWSPricingTerms        `json:"terms"`
}

// AWSProduct represents a purchased SKU
type AWSProduct struct {
	Sku        string               `json:"sku"`
	Attributes AWSProductAttributes `json:"attributes"`
}

// AWSProductAttributes represents metadata about the product used to map to a node.
type AWSProductAttributes struct {
	Location        string `json:"location"`
	InstanceType    string `json:"instanceType"`
	Memory          string `json:"memory"`
	Storage         string `json:"storage"`
	VCpu            string `json:"vcpu"`
	UsageType       string `json:"usagetype"`
	OperatingSystem string `json:"operatingSystem"`
	PreInstalledSw  string `json:"preInstalledSw"`
	InstanceFamily  string `json:"instanceFamily"`
	CapacityStatus  string `json:"capacitystatus"`
	GPU             string `json:"gpu"` // GPU represents the number of GPU on the instance
}

// AWSPricingTerms are how you pay for the node: OnDemand, Reserved, or (TODO) Spot
type AWSPricingTerms struct {
	OnDemand map[string]map[string]*AWSOfferTerm `json:"OnDemand"`
	Reserved map[string]map[string]*AWSOfferTerm `json:"Reserved"`
}

// AWSOfferTerm is a sku extension used to pay for the node.
type AWSOfferTerm struct {
	Sku             string                  `json:"sku"`
	PriceDimensions map[string]*AWSRateCode `json:"priceDimensions"`
}

func (ot *AWSOfferTerm) String() string {
	var strs []string
	for k, rc := range ot.PriceDimensions {
		strs = append(strs, fmt.Sprintf("%s:%s", k, rc.String()))
	}
	return fmt.Sprintf("%s:%s", ot.Sku, strings.Join(strs, ","))
}

// AWSRateCode encodes data about the price of a product
type AWSRateCode struct {
	Unit         string          `json:"unit"`
	PricePerUnit AWSCurrencyCode `json:"pricePerUnit"`
}

func (rc *AWSRateCode) String() string {
	return fmt.Sprintf("{unit: %s, pricePerUnit: %v", rc.Unit, rc.PricePerUnit)
}

// AWSCurrencyCode is the localized currency. (TODO: support non-USD)
type AWSCurrencyCode struct {
	USD string `json:"USD,omitempty"`
	CNY string `json:"CNY,omitempty"`
}

// AWSProductTerms represents the full terms of the product
type AWSProductTerms struct {
	Sku      string        `json:"sku"`
	OnDemand *AWSOfferTerm `json:"OnDemand"`
	Reserved *AWSOfferTerm `json:"Reserved"`
	Memory   string        `json:"memory"`
	Storage  string        `json:"storage"`
	VCpu     string        `json:"vcpu"`
	GPU      string        `json:"gpu"` // GPU represents the number of GPU on the instance
	PV       *PV           `json:"pv"`
}

// ClusterIdEnvVar is the environment variable in which one can manually set the ClusterId
const ClusterIdEnvVar = "AWS_CLUSTER_ID"

// OnDemandRateCode is appended to an node sku
const OnDemandRateCode = ".JRTCKXETXF"
const OnDemandRateCodeCn = ".99YE2YK9UR"

// ReservedRateCode is appended to a node sku
const ReservedRateCode = ".38NPMPTW36"

// HourlyRateCode is appended to a node sku
const HourlyRateCode = ".6YS6EN2CT7"
const HourlyRateCodeCn = ".Q7UJUT2CE6"

// volTypes are used to map between AWS UsageTypes and
// EBS volume types, as they would appear in K8s storage class
// name and the EC2 API.
var volTypes = map[string]string{
	"EBS:VolumeUsage.gp2":    "gp2",
	"EBS:VolumeUsage.gp3":    "gp3",
	"EBS:VolumeUsage":        "standard",
	"EBS:VolumeUsage.sc1":    "sc1",
	"EBS:VolumeP-IOPS.piops": "io1",
	"EBS:VolumeUsage.st1":    "st1",
	"EBS:VolumeUsage.piops":  "io1",
	"gp2":                    "EBS:VolumeUsage.gp2",
	"gp3":                    "EBS:VolumeUsage.gp3",
	"standard":               "EBS:VolumeUsage",
	"sc1":                    "EBS:VolumeUsage.sc1",
	"io1":                    "EBS:VolumeUsage.piops",
	"st1":                    "EBS:VolumeUsage.st1",
}

// locationToRegion maps AWS region names (As they come from Billing)
// to actual region identifiers
var locationToRegion = map[string]string{
	"US East (Ohio)":            "us-east-2",
	"US East (N. Virginia)":     "us-east-1",
	"US West (N. California)":   "us-west-1",
	"US West (Oregon)":          "us-west-2",
	"Asia Pacific (Hong Kong)":  "ap-east-1",
	"Asia Pacific (Mumbai)":     "ap-south-1",
	"Asia Pacific (Osaka)":      "ap-northeast-3",
	"Asia Pacific (Seoul)":      "ap-northeast-2",
	"Asia Pacific (Singapore)":  "ap-southeast-1",
	"Asia Pacific (Sydney)":     "ap-southeast-2",
	"Asia Pacific (Tokyo)":      "ap-northeast-1",
	"Asia Pacific (Jakarta)":    "ap-southeast-3",
	"Canada (Central)":          "ca-central-1",
	"China (Beijing)":           "cn-north-1",
	"China (Ningxia)":           "cn-northwest-1",
	"EU (Frankfurt)":            "eu-central-1",
	"EU (Ireland)":              "eu-west-1",
	"EU (London)":               "eu-west-2",
	"EU (Paris)":                "eu-west-3",
	"EU (Stockholm)":            "eu-north-1",
	"EU (Milan)":                "eu-south-1",
	"South America (Sao Paulo)": "sa-east-1",
	"Africa (Cape Town)":        "af-south-1",
	"AWS GovCloud (US-East)":    "us-gov-east-1",
	"AWS GovCloud (US-West)":    "us-gov-west-1",
}

var regionToBillingRegionCode = map[string]string{
	"us-east-2":      "USE2",
	"us-east-1":      "",
	"us-west-1":      "USW1",
	"us-west-2":      "USW2",
	"ap-east-1":      "APE1",
	"ap-south-1":     "APS3",
	"ap-northeast-3": "APN3",
	"ap-northeast-2": "APN2",
	"ap-southeast-1": "APS1",
	"ap-southeast-2": "APS2",
	"ap-northeast-1": "APN1",
	"ap-southeast-3": "APS4",
	"ca-central-1":   "CAN1",
	"cn-north-1":     "",
	"cn-northwest-1": "",
	"eu-central-1":   "EUC1",
	"eu-west-1":      "EU",
	"eu-west-2":      "EUW2",
	"eu-west-3":      "EUW3",
	"eu-north-1":     "EUN1",
	"eu-south-1":     "EUS1",
	"sa-east-1":      "SAE1",
	"af-south-1":     "AFS1",
	"us-gov-east-1":  "UGE1",
	"us-gov-west-1":  "UGW1",
}

var loadedAWSSecret bool = false
var awsSecret *AWSAccessKey = nil

func (aws *AWS) GetLocalStorageQuery(window, offset time.Duration, rate bool, used bool) string {
	return ""
}

// KubeAttrConversion maps the k8s labels for region to an aws region
func (aws *AWS) KubeAttrConversion(location, instanceType, operatingSystem string) string {
	operatingSystem = strings.ToLower(operatingSystem)

	region := locationToRegion[location]
	return region + "," + instanceType + "," + operatingSystem
}

// AwsSpotFeedInfo contains configuration for spot feed integration
type AwsSpotFeedInfo struct {
	BucketName       string `json:"bucketName"`
	Prefix           string `json:"prefix"`
	Region           string `json:"region"`
	AccountID        string `json:"projectID"`
	ServiceKeyName   string `json:"serviceKeyName"`
	ServiceKeySecret string `json:"serviceKeySecret"`
	SpotLabel        string `json:"spotLabel"`
	SpotLabelValue   string `json:"spotLabelValue"`
}

// AwsAthenaInfo contains configuration for CUR integration
type AwsAthenaInfo struct {
	AthenaBucketName string `json:"athenaBucketName"`
	AthenaRegion     string `json:"athenaRegion"`
	AthenaDatabase   string `json:"athenaDatabase"`
	AthenaTable      string `json:"athenaTable"`
	AthenaWorkgroup  string `json:"athenaWorkgroup"`
	ServiceKeyName   string `json:"serviceKeyName"`
	ServiceKeySecret string `json:"serviceKeySecret"`
	AccountID        string `json:"projectID"`
	MasterPayerARN   string `json:"masterPayerARN"`
}

// IsEmpty returns true if all fields in config are empty, false if not.
func (aai *AwsAthenaInfo) IsEmpty() bool {
	return aai.AthenaBucketName == "" &&
		aai.AthenaRegion == "" &&
		aai.AthenaDatabase == "" &&
		aai.AthenaTable == "" &&
		aai.AthenaWorkgroup == "" &&
		aai.ServiceKeyName == "" &&
		aai.ServiceKeySecret == "" &&
		aai.AccountID == "" &&
		aai.MasterPayerARN == ""
}

// CreateConfig creates an AWS SDK V2 Config for the credentials that it contains
func (aai *AwsAthenaInfo) CreateConfig() (awsSDK.Config, error) {
	keyProvider := AWSAccessKey{AccessKeyID: aai.ServiceKeyName, SecretAccessKey: aai.ServiceKeySecret}
	cfg, err := keyProvider.CreateConfig(aai.AthenaRegion)
	if err != nil {
		return cfg, err
	}
	if aai.MasterPayerARN != "" {
		// Create the credentials from AssumeRoleProvider to assume the role
		// referenced by the roleARN.
		stsSvc := sts.NewFromConfig(cfg)
		creds := stscreds.NewAssumeRoleProvider(stsSvc, aai.MasterPayerARN)
		cfg.Credentials = awsSDK.NewCredentialsCache(creds)
	}
	return cfg, nil
}

func (aws *AWS) GetManagementPlatform() (string, error) {
	nodes := aws.Clientset.GetAllNodes()

	if len(nodes) > 0 {
		n := nodes[0]
		version := n.Status.NodeInfo.KubeletVersion
		if strings.Contains(version, "eks") {
			return "eks", nil
		}
		if _, ok := n.Labels["kops.k8s.io/instancegroup"]; ok {
			return "kops", nil
		}
	}
	return "", nil
}

func (aws *AWS) GetConfig() (*CustomPricing, error) {
	c, err := aws.Config.GetCustomPricingData()
	if err != nil {
		return nil, err
	}
	if c.Discount == "" {
		c.Discount = "0%"
	}
	if c.NegotiatedDiscount == "" {
		c.NegotiatedDiscount = "0%"
	}
	if c.ShareTenancyCosts == "" {
		c.ShareTenancyCosts = defaultShareTenancyCost
	}

	return c, nil
}

// GetAWSAccessKey generate an AWSAccessKey object from the config
func (aws *AWS) GetAWSAccessKey() (*AWSAccessKey, error) {
	config, err := aws.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve AwsAthenaInfo %s", err)
	}
	err = aws.ConfigureAuthWith(config)
	if err != nil {
		return nil, fmt.Errorf("error configuring Cloud Provider %s", err)
	}
	//Look for service key values in env if not present in config
	if config.ServiceKeyName == "" {
		config.ServiceKeyName = env.GetAWSAccessKeyID()
	}
	if config.ServiceKeySecret == "" {
		config.ServiceKeySecret = env.GetAWSAccessKeySecret()
	}

	if config.ServiceKeyName == "" && config.ServiceKeySecret == "" {
		log.DedupedInfof(1, "missing service key values for AWS cloud integration attempting to use service account integration")
	}

	return &AWSAccessKey{AccessKeyID: config.ServiceKeyName, SecretAccessKey: config.ServiceKeySecret}, nil
}

// GetAWSAthenaInfo generate an AWSAthenaInfo object from the config
func (aws *AWS) GetAWSAthenaInfo() (*AwsAthenaInfo, error) {
	config, err := aws.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve AwsAthenaInfo %s", err)
	}

	aak, err := aws.GetAWSAccessKey()
	if err != nil {
		return nil, err
	}

	return &AwsAthenaInfo{
		AthenaBucketName: config.AthenaBucketName,
		AthenaRegion:     config.AthenaRegion,
		AthenaDatabase:   config.AthenaDatabase,
		AthenaTable:      config.AthenaTable,
		AthenaWorkgroup:  config.AthenaWorkgroup,
		ServiceKeyName:   aak.AccessKeyID,
		ServiceKeySecret: aak.SecretAccessKey,
		AccountID:        config.AthenaProjectID,
		MasterPayerARN:   config.MasterPayerARN,
	}, nil
}

func (aws *AWS) UpdateConfigFromConfigMap(cm map[string]string) (*CustomPricing, error) {
	return aws.Config.UpdateFromMap(cm)
}

func (aws *AWS) UpdateConfig(r io.Reader, updateType string) (*CustomPricing, error) {
	return aws.Config.Update(func(c *CustomPricing) error {
		if updateType == SpotInfoUpdateType {
			asfi := AwsSpotFeedInfo{}
			err := json.NewDecoder(r).Decode(&asfi)
			if err != nil {
				return err
			}

			c.ServiceKeyName = asfi.ServiceKeyName
			if asfi.ServiceKeySecret != "" {
				c.ServiceKeySecret = asfi.ServiceKeySecret
			}
			c.SpotDataPrefix = asfi.Prefix
			c.SpotDataBucket = asfi.BucketName
			c.ProjectID = asfi.AccountID
			c.SpotDataRegion = asfi.Region
			c.SpotLabel = asfi.SpotLabel
			c.SpotLabelValue = asfi.SpotLabelValue

		} else if updateType == AthenaInfoUpdateType {
			aai := AwsAthenaInfo{}
			err := json.NewDecoder(r).Decode(&aai)
			if err != nil {
				return err
			}
			c.AthenaBucketName = aai.AthenaBucketName
			c.AthenaRegion = aai.AthenaRegion
			c.AthenaDatabase = aai.AthenaDatabase
			c.AthenaTable = aai.AthenaTable
			c.AthenaWorkgroup = aai.AthenaWorkgroup
			c.ServiceKeyName = aai.ServiceKeyName
			if aai.ServiceKeySecret != "" {
				c.ServiceKeySecret = aai.ServiceKeySecret
			}
			if aai.MasterPayerARN != "" {
				c.MasterPayerARN = aai.MasterPayerARN
			}
			c.AthenaProjectID = aai.AccountID
		} else {
			a := make(map[string]interface{})
			err := json.NewDecoder(r).Decode(&a)
			if err != nil {
				return err
			}
			for k, v := range a {
				kUpper := strings.Title(k) // Just so we consistently supply / receive the same values, uppercase the first letter.
				vstr, ok := v.(string)
				if ok {
					err := SetCustomPricingField(c, kUpper, vstr)
					if err != nil {
						return err
					}
				} else {
					return fmt.Errorf("type error while updating config for %s", kUpper)
				}
			}
		}

		if env.IsRemoteEnabled() {
			err := UpdateClusterMeta(env.GetClusterID(), c.ClusterName)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

type awsKey struct {
	SpotLabelName  string
	SpotLabelValue string
	Labels         map[string]string
	ProviderID     string
}

func (k *awsKey) GPUCount() int {
	return 0
}

func (k *awsKey) GPUType() string {
	return ""
}

func (k *awsKey) ID() string {
	for matchNum, group := range provIdRx.FindStringSubmatch(k.ProviderID) {
		if matchNum == 2 {
			return group
		}
	}
	log.Warnf("Could not find instance ID in \"%s\"", k.ProviderID)
	return ""
}

func (k *awsKey) Features() string {

	instanceType, _ := util.GetInstanceType(k.Labels)
	operatingSystem, _ := util.GetOperatingSystem(k.Labels)
	region, _ := util.GetRegion(k.Labels)

	key := region + "," + instanceType + "," + operatingSystem
	usageType := PreemptibleType
	spotKey := key + "," + usageType
	if l, ok := k.Labels["lifecycle"]; ok && l == "EC2Spot" {
		return spotKey
	}
	if l, ok := k.Labels[k.SpotLabelName]; ok && l == k.SpotLabelValue {
		return spotKey
	}
	return key
}

func (aws *AWS) PVPricing(pvk PVKey) (*PV, error) {
	pricing, ok := aws.Pricing[pvk.Features()]
	if !ok {
		log.Debugf("Persistent Volume pricing not found for %s: %s", pvk.GetStorageClass(), pvk.Features())
		return &PV{}, nil
	}
	return pricing.PV, nil
}

type awsPVKey struct {
	Labels                 map[string]string
	StorageClassParameters map[string]string
	StorageClassName       string
	Name                   string
	DefaultRegion          string
	ProviderID             string
}

func (aws *AWS) GetPVKey(pv *v1.PersistentVolume, parameters map[string]string, defaultRegion string) PVKey {
	providerID := ""
	if pv.Spec.AWSElasticBlockStore != nil {
		providerID = pv.Spec.AWSElasticBlockStore.VolumeID
	} else if pv.Spec.CSI != nil {
		providerID = pv.Spec.CSI.VolumeHandle
	}
	return &awsPVKey{
		Labels:                 pv.Labels,
		StorageClassName:       pv.Spec.StorageClassName,
		StorageClassParameters: parameters,
		Name:                   pv.Name,
		DefaultRegion:          defaultRegion,
		ProviderID:             providerID,
	}
}

func (key *awsPVKey) ID() string {
	return key.ProviderID
}

func (key *awsPVKey) GetStorageClass() string {
	return key.StorageClassName
}

func (key *awsPVKey) Features() string {
	storageClass := key.StorageClassParameters["type"]
	if storageClass == "standard" {
		storageClass = "gp2"
	}
	// Storage class names are generally EBS volume types (gp2)
	// Keys in Pricing are based on UsageTypes (EBS:VolumeType.gp2)
	// Converts between the 2
	region, ok := util.GetRegion(key.Labels)
	if !ok {
		region = key.DefaultRegion
	}
	class, ok := volTypes[storageClass]
	if !ok {
		log.Debugf("No voltype mapping for %s's storageClass: %s", key.Name, storageClass)
	}
	return region + "," + class
}

// GetKey maps node labels to information needed to retrieve pricing data
func (aws *AWS) GetKey(labels map[string]string, n *v1.Node) Key {
	return &awsKey{
		SpotLabelName:  aws.SpotLabelName,
		SpotLabelValue: aws.SpotLabelValue,
		Labels:         labels,
		ProviderID:     labels["providerID"],
	}
}

func (aws *AWS) isPreemptible(key string) bool {
	s := strings.Split(key, ",")
	if len(s) == 4 && s[3] == PreemptibleType {
		return true
	}
	return false
}

func (aws *AWS) ClusterManagementPricing() (string, float64, error) {
	return aws.clusterProvisioner, aws.clusterManagementPrice, nil
}

// Use the pricing data from the current region. Fall back to using all region data if needed.
func (aws *AWS) getRegionPricing(nodeList []*v1.Node) (*http.Response, string, error) {

	pricingURL := "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonEC2/current/"
	region := ""
	multiregion := false
	for _, n := range nodeList {
		labels := n.GetLabels()
		currentNodeRegion := ""
		if r, ok := util.GetRegion(labels); ok {
			currentNodeRegion = r
			// Switch to Chinese endpoint for regions with the Chinese prefix
			if strings.HasPrefix(currentNodeRegion, "cn-") {
				pricingURL = "https://pricing.cn-north-1.amazonaws.com.cn/offers/v1.0/cn/AmazonEC2/current/"
			}
		} else {
			multiregion = true // We weren't able to detect the node's region, so pull all data.
			break
		}
		if region == "" { // We haven't set a region yet
			region = currentNodeRegion
		} else if region != "" && currentNodeRegion != region { // If two nodes have different regions here, we'll need to fetch all pricing data.
			multiregion = true
			break
		}
	}

	// Chinese multiregion endpoint only contains data for Chinese regions and Chinese regions are excluded from other endpoint
	if region != "" && !multiregion {
		pricingURL += region + "/"
	}

	pricingURL += "index.json"

	if env.GetAWSPricingURL() != "" { // Allow override of pricing URL
		pricingURL = env.GetAWSPricingURL()
	}

	log.Infof("starting download of \"%s\", which is quite large ...", pricingURL)
	resp, err := http.Get(pricingURL)
	if err != nil {
		log.Errorf("Bogus fetch of \"%s\": %v", pricingURL, err)
		return nil, pricingURL, err
	}
	return resp, pricingURL, err
}

// SpotRefreshEnabled determines whether the required configs to run the spot feed query have been set up
func (aws *AWS) SpotRefreshEnabled() bool {
	// Need a valid value for at least one of these fields to consider spot pricing as enabled
	return len(aws.SpotDataBucket) != 0 || len(aws.SpotDataRegion) != 0 || len(aws.ProjectID) != 0
}

// DownloadPricingData fetches data from the AWS Pricing API
func (aws *AWS) DownloadPricingData() error {
	aws.DownloadPricingDataLock.Lock()
	defer aws.DownloadPricingDataLock.Unlock()
	c, err := aws.Config.GetCustomPricingData()
	if err != nil {
		log.Errorf("Error downloading default pricing data: %s", err.Error())
	}
	aws.BaseCPUPrice = c.CPU
	aws.BaseRAMPrice = c.RAM
	aws.BaseGPUPrice = c.GPU
	aws.BaseSpotCPUPrice = c.SpotCPU
	aws.BaseSpotRAMPrice = c.SpotRAM
	aws.BaseSpotGPUPrice = c.SpotGPU
	aws.SpotLabelName = c.SpotLabel
	aws.SpotLabelValue = c.SpotLabelValue
	aws.SpotDataBucket = c.SpotDataBucket
	aws.SpotDataPrefix = c.SpotDataPrefix
	aws.ProjectID = c.ProjectID
	aws.SpotDataRegion = c.SpotDataRegion

	aws.ConfigureAuthWith(c) // load aws authentication from configuration or secret

	if len(aws.SpotDataBucket) != 0 && len(aws.ProjectID) == 0 {
		log.Warnf("using SpotDataBucket \"%s\" without ProjectID will not end well", aws.SpotDataBucket)
	}
	nodeList := aws.Clientset.GetAllNodes()

	inputkeys := make(map[string]bool)
	for _, n := range nodeList {
		if _, ok := n.Labels["eks.amazonaws.com/nodegroup"]; ok {
			aws.clusterManagementPrice = 0.10
			aws.clusterProvisioner = "EKS"
		} else if _, ok := n.Labels["kops.k8s.io/instancegroup"]; ok {
			aws.clusterProvisioner = "KOPS"
		}

		labels := n.GetObjectMeta().GetLabels()
		key := aws.GetKey(labels, n)
		inputkeys[key.Features()] = true
	}

	pvList := aws.Clientset.GetAllPersistentVolumes()

	storageClasses := aws.Clientset.GetAllStorageClasses()
	storageClassMap := make(map[string]map[string]string)
	for _, storageClass := range storageClasses {
		params := storageClass.Parameters
		storageClassMap[storageClass.ObjectMeta.Name] = params
		if storageClass.GetAnnotations()["storageclass.kubernetes.io/is-default-class"] == "true" || storageClass.GetAnnotations()["storageclass.beta.kubernetes.io/is-default-class"] == "true" {
			storageClassMap["default"] = params
			storageClassMap[""] = params
		}
	}

	pvkeys := make(map[string]PVKey)
	for _, pv := range pvList {
		params, ok := storageClassMap[pv.Spec.StorageClassName]
		if !ok {
			log.Infof("Unable to find params for storageClassName %s, falling back to default pricing", pv.Spec.StorageClassName)
			continue
		}
		key := aws.GetPVKey(pv, params, "")
		pvkeys[key.Features()] = key
	}

	// RIDataRunning establishes the existence of the goroutine. Since it's possible we
	// run multiple downloads, we don't want to create multiple go routines if one already exists
	if !aws.RIDataRunning {
		err = aws.GetReservationDataFromAthena() // Block until one run has completed.
		if err != nil {
			log.Errorf("Failed to lookup reserved instance data: %s", err.Error())
		} else { // If we make one successful run, check on new reservation data every hour
			go func() {
				defer errs.HandlePanic()
				aws.RIDataRunning = true

				for {
					log.Infof("Reserved Instance watcher running... next update in 1h")
					time.Sleep(time.Hour)
					err := aws.GetReservationDataFromAthena()
					if err != nil {
						log.Infof("Error updating RI data: %s", err.Error())
					}
				}
			}()
		}
	}
	if !aws.SavingsPlanDataRunning {
		err = aws.GetSavingsPlanDataFromAthena()
		if err != nil {
			log.Errorf("Failed to lookup savings plan data: %s", err.Error())
		} else {
			go func() {
				defer errs.HandlePanic()
				aws.SavingsPlanDataRunning = true
				for {
					log.Infof("Savings Plan watcher running... next update in 1h")
					time.Sleep(time.Hour)
					err := aws.GetSavingsPlanDataFromAthena()
					if err != nil {
						log.Infof("Error updating Savings Plan data: %s", err.Error())
					}
				}
			}()
		}
	}

	aws.Pricing = make(map[string]*AWSProductTerms)
	aws.ValidPricingKeys = make(map[string]bool)
	skusToKeys := make(map[string]string)

	resp, pricingURL, err := aws.getRegionPricing(nodeList)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(resp.Body)
	for {
		t, err := dec.Token()
		if err == io.EOF {
			log.Infof("done loading \"%s\"\n", pricingURL)
			break
		} else if err != nil {
			log.Errorf("error parsing response json %v", resp.Body)
			break
		}
		if t == "products" {
			_, err := dec.Token() // this should parse the opening "{""
			if err != nil {
				return err
			}
			for dec.More() {
				_, err := dec.Token() // the sku token
				if err != nil {
					return err
				}
				product := &AWSProduct{}

				err = dec.Decode(&product)
				if err != nil {
					log.Errorf("Error parsing response from \"%s\": %v", pricingURL, err.Error())
					break
				}

				if product.Attributes.PreInstalledSw == "NA" &&
					(strings.HasPrefix(product.Attributes.UsageType, "BoxUsage") || strings.Contains(product.Attributes.UsageType, "-BoxUsage")) &&
					product.Attributes.CapacityStatus == "Used" {
					key := aws.KubeAttrConversion(product.Attributes.Location, product.Attributes.InstanceType, product.Attributes.OperatingSystem)
					spotKey := key + ",preemptible"
					if inputkeys[key] || inputkeys[spotKey] { // Just grab the sku even if spot, and change the price later.
						productTerms := &AWSProductTerms{
							Sku:     product.Sku,
							Memory:  product.Attributes.Memory,
							Storage: product.Attributes.Storage,
							VCpu:    product.Attributes.VCpu,
							GPU:     product.Attributes.GPU,
						}
						aws.Pricing[key] = productTerms
						aws.Pricing[spotKey] = productTerms
						skusToKeys[product.Sku] = key
					}
					aws.ValidPricingKeys[key] = true
					aws.ValidPricingKeys[spotKey] = true
				} else if strings.Contains(product.Attributes.UsageType, "EBS:Volume") {
					// UsageTypes may be prefixed with a region code - we're removing this when using
					// volTypes to keep lookups generic
					usageTypeMatch := usageTypeRegx.FindStringSubmatch(product.Attributes.UsageType)
					usageTypeNoRegion := usageTypeMatch[len(usageTypeMatch)-1]
					key := locationToRegion[product.Attributes.Location] + "," + usageTypeNoRegion
					spotKey := key + ",preemptible"
					pv := &PV{
						Class:  volTypes[usageTypeNoRegion],
						Region: locationToRegion[product.Attributes.Location],
					}
					productTerms := &AWSProductTerms{
						Sku: product.Sku,
						PV:  pv,
					}
					aws.Pricing[key] = productTerms
					aws.Pricing[spotKey] = productTerms
					skusToKeys[product.Sku] = key
					aws.ValidPricingKeys[key] = true
					aws.ValidPricingKeys[spotKey] = true
				}
			}
		}
		if t == "terms" {
			_, err := dec.Token() // this should parse the opening "{""
			if err != nil {
				return err
			}
			termType, err := dec.Token()
			if err != nil {
				return err
			}
			if termType == "OnDemand" {
				_, err := dec.Token()
				if err != nil { // again, should parse an opening "{"
					return err
				}
				for dec.More() {
					sku, err := dec.Token()
					if err != nil {
						return err
					}
					_, err = dec.Token() // another opening "{"
					if err != nil {
						return err
					}
					skuOnDemand, err := dec.Token()
					if err != nil {
						return err
					}
					offerTerm := &AWSOfferTerm{}
					err = dec.Decode(&offerTerm)
					if err != nil {
						log.Errorf("Error decoding AWS Offer Term: " + err.Error())
					}

					key, ok := skusToKeys[sku.(string)]
					spotKey := key + ",preemptible"
					if ok {
						aws.Pricing[key].OnDemand = offerTerm
						aws.Pricing[spotKey].OnDemand = offerTerm
						var cost string
						if sku.(string)+OnDemandRateCode == skuOnDemand {
							cost = offerTerm.PriceDimensions[sku.(string)+OnDemandRateCode+HourlyRateCode].PricePerUnit.USD
						} else if sku.(string)+OnDemandRateCodeCn == skuOnDemand {
							cost = offerTerm.PriceDimensions[sku.(string)+OnDemandRateCodeCn+HourlyRateCodeCn].PricePerUnit.CNY
						}
						if strings.Contains(key, "EBS:VolumeP-IOPS.piops") {
							// If the specific UsageType is the per IO cost used on io1 volumes
							// we need to add the per IO cost to the io1 PV cost

							// Add the per IO cost to the PV object for the io1 volume type
							aws.Pricing[key].PV.CostPerIO = cost
						} else if strings.Contains(key, "EBS:Volume") {
							// If volume, we need to get hourly cost and add it to the PV object
							costFloat, _ := strconv.ParseFloat(cost, 64)
							hourlyPrice := costFloat / 730

							aws.Pricing[key].PV.Cost = strconv.FormatFloat(hourlyPrice, 'f', -1, 64)
						}
					}

					_, err = dec.Token()
					if err != nil {
						return err
					}
				}
				_, err = dec.Token()
				if err != nil {
					return err
				}
			}
		}
	}
	log.Infof("Finished downloading \"%s\"", pricingURL)

	if !aws.SpotRefreshEnabled() {
		return nil
	}

	// Always run spot pricing refresh when performing download
	aws.refreshSpotPricing(true)

	// Only start a single refresh goroutine
	if !aws.SpotRefreshRunning {
		aws.SpotRefreshRunning = true

		go func() {
			defer errs.HandlePanic()

			for {
				log.Infof("Spot Pricing Refresh scheduled in %.2f minutes.", SpotRefreshDuration.Minutes())
				time.Sleep(SpotRefreshDuration)

				// Reoccurring refresh checks update times
				aws.refreshSpotPricing(false)
			}
		}()
	}

	return nil
}

func (aws *AWS) refreshSpotPricing(force bool) {
	aws.SpotPricingLock.Lock()
	defer aws.SpotPricingLock.Unlock()

	now := time.Now().UTC()
	updateTime := now.Add(-SpotRefreshDuration)

	// Return if there was an update time set and an hour hasn't elapsed
	if !force && aws.SpotPricingUpdatedAt != nil && aws.SpotPricingUpdatedAt.After(updateTime) {
		return
	}

	sp, err := aws.parseSpotData(aws.SpotDataBucket, aws.SpotDataPrefix, aws.ProjectID, aws.SpotDataRegion)
	if err != nil {
		log.Warnf("Skipping AWS spot data download: %s", err.Error())
		aws.SpotPricingError = err
		return
	}
	aws.SpotPricingError = nil

	// update time last updated
	aws.SpotPricingUpdatedAt = &now
	aws.SpotPricingByInstanceID = sp
}

// Stubbed NetworkPricing for AWS. Pull directly from aws.json for now
func (aws *AWS) NetworkPricing() (*Network, error) {
	cpricing, err := aws.Config.GetCustomPricingData()
	if err != nil {
		return nil, err
	}
	znec, err := strconv.ParseFloat(cpricing.ZoneNetworkEgress, 64)
	if err != nil {
		return nil, err
	}
	rnec, err := strconv.ParseFloat(cpricing.RegionNetworkEgress, 64)
	if err != nil {
		return nil, err
	}
	inec, err := strconv.ParseFloat(cpricing.InternetNetworkEgress, 64)
	if err != nil {
		return nil, err
	}

	return &Network{
		ZoneNetworkEgressCost:     znec,
		RegionNetworkEgressCost:   rnec,
		InternetNetworkEgressCost: inec,
	}, nil
}

func (aws *AWS) LoadBalancerPricing() (*LoadBalancer, error) {
	fffrc := 0.025
	afrc := 0.010
	lbidc := 0.008

	numForwardingRules := 1.0
	dataIngressGB := 0.0

	var totalCost float64
	if numForwardingRules < 5 {
		totalCost = fffrc*numForwardingRules + lbidc*dataIngressGB
	} else {
		totalCost = fffrc*5 + afrc*(numForwardingRules-5) + lbidc*dataIngressGB
	}
	return &LoadBalancer{
		Cost: totalCost,
	}, nil
}

// AllNodePricing returns all the billing data fetched.
func (aws *AWS) AllNodePricing() (interface{}, error) {
	aws.DownloadPricingDataLock.RLock()
	defer aws.DownloadPricingDataLock.RUnlock()
	return aws.Pricing, nil
}

func (aws *AWS) spotPricing(instanceID string) (*spotInfo, bool) {
	aws.SpotPricingLock.RLock()
	defer aws.SpotPricingLock.RUnlock()

	info, ok := aws.SpotPricingByInstanceID[instanceID]
	return info, ok
}

func (aws *AWS) reservedInstancePricing(instanceID string) (*RIData, bool) {
	aws.RIDataLock.RLock()
	defer aws.RIDataLock.RUnlock()

	data, ok := aws.RIPricingByInstanceID[instanceID]
	return data, ok
}

func (aws *AWS) savingsPlanPricing(instanceID string) (*SavingsPlanData, bool) {
	aws.SavingsPlanDataLock.RLock()
	defer aws.SavingsPlanDataLock.RUnlock()

	data, ok := aws.SavingsPlanDataByInstanceID[instanceID]
	return data, ok
}

func (aws *AWS) createNode(terms *AWSProductTerms, usageType string, k Key) (*Node, error) {
	key := k.Features()

	if spotInfo, ok := aws.spotPricing(k.ID()); ok {
		var spotcost string
		log.DedupedInfof(5, "Looking up spot data from feed for node %s", k.ID())
		arr := strings.Split(spotInfo.Charge, " ")
		if len(arr) == 2 {
			spotcost = arr[0]
		} else {
			log.Infof("Spot data for node %s is missing", k.ID())
		}
		return &Node{
			Cost:         spotcost,
			VCPU:         terms.VCpu,
			RAM:          terms.Memory,
			GPU:          terms.GPU,
			Storage:      terms.Storage,
			BaseCPUPrice: aws.BaseCPUPrice,
			BaseRAMPrice: aws.BaseRAMPrice,
			BaseGPUPrice: aws.BaseGPUPrice,
			UsageType:    PreemptibleType,
		}, nil
	} else if aws.isPreemptible(key) { // Preemptible but we don't have any data in the pricing report.
		log.DedupedWarningf(5, "Node %s marked preemptible but we have no data in spot feed", k.ID())
		return &Node{
			VCPU:         terms.VCpu,
			VCPUCost:     aws.BaseSpotCPUPrice,
			RAM:          terms.Memory,
			GPU:          terms.GPU,
			Storage:      terms.Storage,
			BaseCPUPrice: aws.BaseCPUPrice,
			BaseRAMPrice: aws.BaseRAMPrice,
			BaseGPUPrice: aws.BaseGPUPrice,
			UsageType:    PreemptibleType,
		}, nil
	} else if sp, ok := aws.savingsPlanPricing(k.ID()); ok {
		strCost := fmt.Sprintf("%f", sp.EffectiveCost)
		return &Node{
			Cost:         strCost,
			VCPU:         terms.VCpu,
			RAM:          terms.Memory,
			GPU:          terms.GPU,
			Storage:      terms.Storage,
			BaseCPUPrice: aws.BaseCPUPrice,
			BaseRAMPrice: aws.BaseRAMPrice,
			BaseGPUPrice: aws.BaseGPUPrice,
			UsageType:    usageType,
		}, nil

	} else if ri, ok := aws.reservedInstancePricing(k.ID()); ok {
		strCost := fmt.Sprintf("%f", ri.EffectiveCost)
		return &Node{
			Cost:         strCost,
			VCPU:         terms.VCpu,
			RAM:          terms.Memory,
			GPU:          terms.GPU,
			Storage:      terms.Storage,
			BaseCPUPrice: aws.BaseCPUPrice,
			BaseRAMPrice: aws.BaseRAMPrice,
			BaseGPUPrice: aws.BaseGPUPrice,
			UsageType:    usageType,
		}, nil

	}
	var cost string
	c, ok := terms.OnDemand.PriceDimensions[terms.Sku+OnDemandRateCode+HourlyRateCode]
	if ok {
		cost = c.PricePerUnit.USD
	} else {
		// Check for Chinese pricing before throwing error
		c, ok = terms.OnDemand.PriceDimensions[terms.Sku+OnDemandRateCodeCn+HourlyRateCodeCn]
		if ok {
			cost = c.PricePerUnit.CNY
		} else {
			return nil, fmt.Errorf("Could not fetch data for \"%s\"", k.ID())
		}
	}

	return &Node{
		Cost:         cost,
		VCPU:         terms.VCpu,
		RAM:          terms.Memory,
		GPU:          terms.GPU,
		Storage:      terms.Storage,
		BaseCPUPrice: aws.BaseCPUPrice,
		BaseRAMPrice: aws.BaseRAMPrice,
		BaseGPUPrice: aws.BaseGPUPrice,
		UsageType:    usageType,
	}, nil
}

// NodePricing takes in a key from GetKey and returns a Node object for use in building the cost model.
func (aws *AWS) NodePricing(k Key) (*Node, error) {
	aws.DownloadPricingDataLock.RLock()
	defer aws.DownloadPricingDataLock.RUnlock()

	key := k.Features()
	usageType := "ondemand"
	if aws.isPreemptible(key) {
		usageType = PreemptibleType
	}

	terms, ok := aws.Pricing[key]
	if ok {
		return aws.createNode(terms, usageType, k)
	} else if _, ok := aws.ValidPricingKeys[key]; ok {
		aws.DownloadPricingDataLock.RUnlock()
		err := aws.DownloadPricingData()
		aws.DownloadPricingDataLock.RLock()
		if err != nil {
			return &Node{
				Cost:             aws.BaseCPUPrice,
				BaseCPUPrice:     aws.BaseCPUPrice,
				BaseRAMPrice:     aws.BaseRAMPrice,
				BaseGPUPrice:     aws.BaseGPUPrice,
				UsageType:        usageType,
				UsesBaseCPUPrice: true,
			}, err
		}
		terms, termsOk := aws.Pricing[key]
		if !termsOk {
			return &Node{
				Cost:             aws.BaseCPUPrice,
				BaseCPUPrice:     aws.BaseCPUPrice,
				BaseRAMPrice:     aws.BaseRAMPrice,
				BaseGPUPrice:     aws.BaseGPUPrice,
				UsageType:        usageType,
				UsesBaseCPUPrice: true,
			}, fmt.Errorf("Unable to find any Pricing data for \"%s\"", key)
		}
		return aws.createNode(terms, usageType, k)
	} else { // Fall back to base pricing if we can't find the key. Base pricing is handled at the costmodel level.
		return nil, fmt.Errorf("Invalid Pricing Key \"%s\"", key)

	}
}

// ClusterInfo returns an object that represents the cluster. TODO: actually return the name of the cluster. Blocked on cluster federation.
func (awsProvider *AWS) ClusterInfo() (map[string]string, error) {
	defaultClusterName := "AWS Cluster #1"
	c, err := awsProvider.GetConfig()
	if err != nil {
		return nil, err
	}

	remoteEnabled := env.IsRemoteEnabled()

	makeStructure := func(clusterName string) (map[string]string, error) {
		m := make(map[string]string)
		m["name"] = clusterName
		m["provider"] = kubecost.AWSProvider
		m["account"] = c.AthenaProjectID // this value requires configuration but is unavailable else where
		m["region"] = awsProvider.clusterRegion
		m["id"] = env.GetClusterID()
		m["remoteReadEnabled"] = strconv.FormatBool(remoteEnabled)
		m["provisioner"] = awsProvider.clusterProvisioner
		return m, nil
	}

	if c.ClusterName != "" {
		return makeStructure(c.ClusterName)
	}

	maybeClusterId := env.GetAWSClusterID()
	if len(maybeClusterId) != 0 {
		log.Infof("Returning \"%s\" as ClusterName", maybeClusterId)
		return makeStructure(maybeClusterId)
	}

	log.Infof("Unable to sniff out cluster ID, perhaps set $%s to force one", env.AWSClusterIDEnvVar)
	return makeStructure(defaultClusterName)
}

// updates the authentication to the latest values (via config or secret)
func (aws *AWS) ConfigureAuth() error {
	c, err := aws.Config.GetCustomPricingData()
	if err != nil {
		log.Errorf("Error downloading default pricing data: %s", err.Error())
	}
	return aws.ConfigureAuthWith(c)
}

// updates the authentication to the latest values (via config or secret)
func (aws *AWS) ConfigureAuthWith(config *CustomPricing) error {
	accessKeyID, accessKeySecret := aws.getAWSAuth(false, config)
	if accessKeyID != "" && accessKeySecret != "" { // credentials may exist on the actual AWS node-- if so, use those. If not, override with the service key
		err := env.Set(env.AWSAccessKeyIDEnvVar, accessKeyID)
		if err != nil {
			return err
		}
		err = env.Set(env.AWSAccessKeySecretEnvVar, accessKeySecret)
		if err != nil {
			return err
		}
	}
	return nil
}

// Gets the aws key id and secret
func (aws *AWS) getAWSAuth(forceReload bool, cp *CustomPricing) (string, string) {

	// 1. Check config values first (set from frontend UI)
	if cp.ServiceKeyName != "" && cp.ServiceKeySecret != "" {
		aws.serviceAccountChecks.set("hasKey", &ServiceAccountCheck{
			Message: "AWS ServiceKey exists",
			Status:  true,
		})
		return cp.ServiceKeyName, cp.ServiceKeySecret
	}

	// 2. Check for secret
	s, _ := aws.loadAWSAuthSecret(forceReload)
	if s != nil && s.AccessKeyID != "" && s.SecretAccessKey != "" {
		aws.serviceAccountChecks.set("hasKey", &ServiceAccountCheck{
			Message: "AWS ServiceKey exists",
			Status:  true,
		})
		return s.AccessKeyID, s.SecretAccessKey
	}

	// 3. Fall back to env vars
	if env.GetAWSAccessKeyID() == "" || env.GetAWSAccessKeyID() == "" {
		aws.serviceAccountChecks.set("hasKey", &ServiceAccountCheck{
			Message: "AWS ServiceKey exists",
			Status:  false,
		})
	} else {
		aws.serviceAccountChecks.set("hasKey", &ServiceAccountCheck{
			Message: "AWS ServiceKey exists",
			Status:  true,
		})
	}
	return env.GetAWSAccessKeyID(), env.GetAWSAccessKeySecret()
}

// Load once and cache the result (even on failure). This is an install time secret, so
// we don't expect the secret to change. If it does, however, we can force reload using
// the input parameter.
func (aws *AWS) loadAWSAuthSecret(force bool) (*AWSAccessKey, error) {
	if !force && loadedAWSSecret {
		return awsSecret, nil
	}
	loadedAWSSecret = true

	exists, err := fileutil.FileExists(authSecretPath)
	if !exists || err != nil {
		return nil, fmt.Errorf("Failed to locate service account file: %s", authSecretPath)
	}

	result, err := os.ReadFile(authSecretPath)
	if err != nil {
		return nil, err
	}

	var ak AWSAccessKey
	err = json.Unmarshal(result, &ak)
	if err != nil {
		return nil, err
	}

	awsSecret = &ak
	return awsSecret, nil
}

func (aws *AWS) getAddressesForRegion(ctx context.Context, region string) (*ec2.DescribeAddressesOutput, error) {
	aak, err := aws.GetAWSAccessKey()
	if err != nil {
		return nil, err
	}
	cfg, err := aak.CreateConfig(region)
	if err != nil {
		return nil, err
	}

	cli := ec2.NewFromConfig(cfg)
	return cli.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
}

// GetAddresses retrieves EC2 addresses
func (aws *AWS) GetAddresses() ([]byte, error) {
	aws.ConfigureAuth() // load authentication data into env vars

	addressCh := make(chan *ec2.DescribeAddressesOutput, len(awsRegions))
	errorCh := make(chan error, len(awsRegions))

	var wg sync.WaitGroup
	wg.Add(len(awsRegions))

	// Get volumes from each AWS region
	for _, r := range awsRegions {
		// Fetch IP address response and send results and errors to their
		// respective channels
		go func(region string) {
			defer wg.Done()
			defer errs.HandlePanic()

			// Query for first page of volume results
			resp, err := aws.getAddressesForRegion(context.TODO(), region)
			if err != nil {
				errorCh <- err
				return
			}
			addressCh <- resp
		}(r)
	}

	// Close the result channels after everything has been sent
	go func() {
		defer errs.HandlePanic()

		wg.Wait()
		close(errorCh)
		close(addressCh)
	}()

	var addresses []*ec2Types.Address
	for adds := range addressCh {
		for _, add := range adds.Addresses {
			a := add // duplicate to avoid pointer to iterator
			addresses = append(addresses, &a)
		}

	}

	var errs []error
	for err := range errorCh {
		log.DedupedWarningf(5, "unable to get addresses: %s", err)
		errs = append(errs, err)
	}

	// Return error if no addresses are returned
	if len(errs) > 0 && len(addresses) == 0 {
		return nil, fmt.Errorf("%d error(s) retrieving addresses: %v", len(errs), errs)
	}

	// Format the response this way to match the JSON-encoded formatting of a single response
	// from DescribeAddresss, so that consumers can always expect AWS disk responses to have
	// a "Addresss" key at the top level.
	return json.Marshal(map[string][]*ec2Types.Address{
		"Addresses": addresses,
	})
}

func (aws *AWS) getDisksForRegion(ctx context.Context, region string, maxResults int32, nextToken *string) (*ec2.DescribeVolumesOutput, error) {
	aak, err := aws.GetAWSAccessKey()
	if err != nil {
		return nil, err
	}

	cfg, err := aak.CreateConfig(region)
	if err != nil {
		return nil, err
	}

	cli := ec2.NewFromConfig(cfg)
	return cli.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
		MaxResults: &maxResults,
		NextToken:  nextToken,
	})
}

// GetDisks returns the AWS disks backing PVs. Useful because sometimes k8s will not clean up PVs correctly. Requires a json config in /var/configs with key region.
func (aws *AWS) GetDisks() ([]byte, error) {
	aws.ConfigureAuth() // load authentication data into env vars

	volumeCh := make(chan *ec2.DescribeVolumesOutput, len(awsRegions))
	errorCh := make(chan error, len(awsRegions))

	var wg sync.WaitGroup
	wg.Add(len(awsRegions))

	// Get volumes from each AWS region
	for _, r := range awsRegions {
		// Fetch volume response and send results and errors to their
		// respective channels
		go func(region string) {
			defer wg.Done()
			defer errs.HandlePanic()

			// Query for first page of volume results
			resp, err := aws.getDisksForRegion(context.TODO(), region, 1000, nil)
			if err != nil {
				errorCh <- err
				return
			}
			volumeCh <- resp

			// A NextToken indicates more pages of results. Keep querying
			// until all pages are retrieved.
			for resp.NextToken != nil {
				resp, err = aws.getDisksForRegion(context.TODO(), region, 100, resp.NextToken)
				if err != nil {
					errorCh <- err
					return
				}
				volumeCh <- resp
			}
		}(r)
	}

	// Close the result channels after everything has been sent
	go func() {
		defer errs.HandlePanic()

		wg.Wait()
		close(errorCh)
		close(volumeCh)
	}()

	var volumes []*ec2Types.Volume
	for vols := range volumeCh {
		for _, vol := range vols.Volumes {
			v := vol // duplicate to avoid pointer to iterator
			volumes = append(volumes, &v)
		}
	}

	var errs []error
	for err := range errorCh {
		log.DedupedWarningf(5, "unable to get disks: %s", err)
		errs = append(errs, err)
	}

	// Return error if no volumes are returned
	if len(errs) > 0 && len(volumes) == 0 {
		return nil, fmt.Errorf("%d error(s) retrieving volumes: %v", len(errs), errs)
	}

	// Format the response this way to match the JSON-encoded formatting of a single response
	// from DescribeVolumes, so that consumers can always expect AWS disk responses to have
	// a "Volumes" key at the top level.
	return json.Marshal(map[string][]*ec2Types.Volume{
		"Volumes": volumes,
	})
}

func (*AWS) GetOrphanedResources() ([]OrphanedResource, error) {
	return nil, errors.New("not implemented")
}

// QueryAthenaPaginated executes athena query and processes results.
func (aws *AWS) QueryAthenaPaginated(ctx context.Context, query string, fn func(*athena.GetQueryResultsOutput) bool) error {
	awsAthenaInfo, err := aws.GetAWSAthenaInfo()
	if err != nil {
		return err
	}
	if awsAthenaInfo.AthenaDatabase == "" || awsAthenaInfo.AthenaTable == "" || awsAthenaInfo.AthenaRegion == "" ||
		awsAthenaInfo.AthenaBucketName == "" || awsAthenaInfo.AccountID == "" {
		return fmt.Errorf("QueryAthenaPaginated: athena configuration incomplete")
	}

	queryExecutionCtx := &athenaTypes.QueryExecutionContext{
		Database: awsSDK.String(awsAthenaInfo.AthenaDatabase),
	}

	resultConfiguration := &athenaTypes.ResultConfiguration{
		OutputLocation: awsSDK.String(awsAthenaInfo.AthenaBucketName),
	}
	startQueryExecutionInput := &athena.StartQueryExecutionInput{
		QueryString:           awsSDK.String(query),
		QueryExecutionContext: queryExecutionCtx,
		ResultConfiguration:   resultConfiguration,
	}

	// Only set if there is a value, the default input is nil which defaults to the 'primary' workgroup
	if awsAthenaInfo.AthenaWorkgroup != "" {
		startQueryExecutionInput.WorkGroup = awsSDK.String(awsAthenaInfo.AthenaWorkgroup)
	}

	// Create Athena Client
	cfg, err := awsAthenaInfo.CreateConfig()
	if err != nil {
		log.Errorf("Could not retrieve Athena Configuration: %s", err.Error())
	}
	cli := athena.NewFromConfig(cfg)

	// Query Athena
	startQueryExecutionOutput, err := cli.StartQueryExecution(ctx, startQueryExecutionInput)
	if err != nil {
		return fmt.Errorf("QueryAthenaPaginated: start query error: %s", err.Error())
	}
	err = waitForQueryToComplete(ctx, cli, startQueryExecutionOutput.QueryExecutionId)
	if err != nil {
		return fmt.Errorf("QueryAthenaPaginated: query execution error: %s", err.Error())
	}
	queryResultsInput := &athena.GetQueryResultsInput{
		QueryExecutionId: startQueryExecutionOutput.QueryExecutionId,
	}
	getQueryResultsPaginator := athena.NewGetQueryResultsPaginator(cli, queryResultsInput)
	for getQueryResultsPaginator.HasMorePages() {
		pg, err := getQueryResultsPaginator.NextPage(ctx)
		if err != nil {
			log.Errorf("QueryAthenaPaginated: NextPage error: %s", err.Error())
			continue
		}
		fn(pg)
	}
	return nil
}

func waitForQueryToComplete(ctx context.Context, client *athena.Client, queryExecutionID *string) error {
	inp := &athena.GetQueryExecutionInput{
		QueryExecutionId: queryExecutionID,
	}
	isQueryStillRunning := true
	for isQueryStillRunning {
		qe, err := client.GetQueryExecution(ctx, inp)
		if err != nil {
			return err
		}
		if qe.QueryExecution.Status.State == "SUCCEEDED" {
			isQueryStillRunning = false
			continue
		}
		if qe.QueryExecution.Status.State != "RUNNING" && qe.QueryExecution.Status.State != "QUEUED" {
			return fmt.Errorf("no query results available for query %s", *queryExecutionID)
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}

type SavingsPlanData struct {
	ResourceID     string
	EffectiveCost  float64
	SavingsPlanARN string
	MostRecentDate string
}

func (aws *AWS) GetSavingsPlanDataFromAthena() error {
	cfg, err := aws.GetConfig()
	if err != nil {
		aws.RIPricingError = err
		return err
	}
	if cfg.AthenaBucketName == "" {
		err = fmt.Errorf("No Athena Bucket configured")
		aws.RIPricingError = err
		return err
	}
	if aws.SavingsPlanDataByInstanceID == nil {
		aws.SavingsPlanDataByInstanceID = make(map[string]*SavingsPlanData)
	}
	tNow := time.Now()
	tOneDayAgo := tNow.Add(time.Duration(-25) * time.Hour) // Also get files from one day ago to avoid boundary conditions
	start := tOneDayAgo.Format("2006-01-02")
	end := tNow.Format("2006-01-02")
	// Use Savings Plan Effective Rate as an estimation for cost, assuming the 1h most recent period got a fully loaded savings plan.
	//
	q := `SELECT
		line_item_usage_start_date,
		savings_plan_savings_plan_a_r_n,
		line_item_resource_id,
		savings_plan_savings_plan_rate
	FROM %s as cost_data
	WHERE line_item_usage_start_date BETWEEN date '%s' AND date '%s'
	AND line_item_line_item_type = 'SavingsPlanCoveredUsage' ORDER BY
	line_item_usage_start_date DESC`

	page := 0
	processResults := func(op *athena.GetQueryResultsOutput) bool {
		if op == nil {
			log.Errorf("GetSavingsPlanDataFromAthena: Athena page is nil")
			return false
		} else if op.ResultSet == nil {
			log.Errorf("GetSavingsPlanDataFromAthena: Athena page.ResultSet is nil")
			return false
		}
		aws.SavingsPlanDataLock.Lock()
		aws.SavingsPlanDataByInstanceID = make(map[string]*SavingsPlanData) // Clean out the old data and only report a savingsplan price if its in the most recent run.
		mostRecentDate := ""
		iter := op.ResultSet.Rows
		if page == 0 && len(iter) > 0 {
			iter = op.ResultSet.Rows[1:len(op.ResultSet.Rows)]
		}
		page++
		for _, r := range iter {
			d := *r.Data[0].VarCharValue
			if mostRecentDate == "" {
				mostRecentDate = d
			} else if mostRecentDate != d { // Get all most recent assignments
				break
			}
			cost, err := strconv.ParseFloat(*r.Data[3].VarCharValue, 64)
			if err != nil {
				log.Infof("Error converting `%s` from float ", *r.Data[3].VarCharValue)
			}
			r := &SavingsPlanData{
				ResourceID:     *r.Data[2].VarCharValue,
				EffectiveCost:  cost,
				SavingsPlanARN: *r.Data[1].VarCharValue,
				MostRecentDate: d,
			}
			aws.SavingsPlanDataByInstanceID[r.ResourceID] = r
		}
		log.Debugf("Found %d savings plan applied instances", len(aws.SavingsPlanDataByInstanceID))
		for k, r := range aws.SavingsPlanDataByInstanceID {
			log.DedupedInfof(5, "Savings Plan Instance Data found for node %s : %f at time %s", k, r.EffectiveCost, r.MostRecentDate)
		}
		aws.SavingsPlanDataLock.Unlock()
		return true
	}

	query := fmt.Sprintf(q, cfg.AthenaTable, start, end)

	log.Debugf("Running Query: %s", query)

	err = aws.QueryAthenaPaginated(context.TODO(), query, processResults)
	if err != nil {
		aws.RIPricingError = err
		return fmt.Errorf("Error fetching Savings Plan Data: %s", err)
	}

	return nil
}

type RIData struct {
	ResourceID     string
	EffectiveCost  float64
	ReservationARN string
	MostRecentDate string
}

func (aws *AWS) GetReservationDataFromAthena() error {
	cfg, err := aws.GetConfig()
	if err != nil {
		aws.RIPricingError = err
		return err
	}
	if cfg.AthenaBucketName == "" {
		err = fmt.Errorf("No Athena Bucket configured")
		aws.RIPricingError = err
		return err
	}

	// Query for all column names in advance in order to validate configured
	// label columns
	columns, _ := aws.fetchColumns()

	if !columns["reservation_reservation_a_r_n"] || !columns["reservation_effective_cost"] {
		err = fmt.Errorf("no reservation data available in Athena")
		aws.RIPricingError = err
		return err
	}
	if aws.RIPricingByInstanceID == nil {
		aws.RIPricingByInstanceID = make(map[string]*RIData)
	}
	tNow := time.Now()
	tOneDayAgo := tNow.Add(time.Duration(-25) * time.Hour) // Also get files from one day ago to avoid boundary conditions
	start := tOneDayAgo.Format("2006-01-02")
	end := tNow.Format("2006-01-02")
	q := `SELECT
		line_item_usage_start_date,
		reservation_reservation_a_r_n,
		line_item_resource_id,
		reservation_effective_cost
	FROM %s as cost_data
	WHERE line_item_usage_start_date BETWEEN date '%s' AND date '%s'
	AND reservation_reservation_a_r_n <> '' ORDER BY
	line_item_usage_start_date DESC`

	page := 0
	processResults := func(op *athena.GetQueryResultsOutput) bool {
		if op == nil {
			log.Errorf("GetReservationDataFromAthena: Athena page is nil")
			return false
		} else if op.ResultSet == nil {
			log.Errorf("GetReservationDataFromAthena: Athena page.ResultSet is nil")
			return false
		}
		aws.RIDataLock.Lock()
		aws.RIPricingByInstanceID = make(map[string]*RIData) // Clean out the old data and only report a RI price if its in the most recent run.
		mostRecentDate := ""
		iter := op.ResultSet.Rows
		if page == 0 && len(iter) > 0 {
			iter = op.ResultSet.Rows[1:len(op.ResultSet.Rows)]
		}
		page++
		for _, r := range iter {
			d := *r.Data[0].VarCharValue
			if mostRecentDate == "" {
				mostRecentDate = d
			} else if mostRecentDate != d { // Get all most recent assignments
				break
			}
			cost, err := strconv.ParseFloat(*r.Data[3].VarCharValue, 64)
			if err != nil {
				log.Infof("Error converting `%s` from float ", *r.Data[3].VarCharValue)
			}
			r := &RIData{
				ResourceID:     *r.Data[2].VarCharValue,
				EffectiveCost:  cost,
				ReservationARN: *r.Data[1].VarCharValue,
				MostRecentDate: d,
			}
			aws.RIPricingByInstanceID[r.ResourceID] = r
		}
		log.Debugf("Found %d reserved instances", len(aws.RIPricingByInstanceID))
		for k, r := range aws.RIPricingByInstanceID {
			log.DedupedInfof(5, "Reserved Instance Data found for node %s : %f at time %s", k, r.EffectiveCost, r.MostRecentDate)
		}
		aws.RIDataLock.Unlock()
		return true
	}

	query := fmt.Sprintf(q, cfg.AthenaTable, start, end)

	log.Debugf("Running Query: %s", query)

	err = aws.QueryAthenaPaginated(context.TODO(), query, processResults)
	if err != nil {
		aws.RIPricingError = err
		return fmt.Errorf("Error fetching Reserved Instance Data: %s", err)
	}
	aws.RIPricingError = nil
	return nil
}

// fetchColumns returns a list of the names of all columns in the configured
// Athena tables
func (aws *AWS) fetchColumns() (map[string]bool, error) {
	columnSet := map[string]bool{}

	awsAthenaInfo, err := aws.GetAWSAthenaInfo()
	if err != nil {
		return nil, err
	}

	// This Query is supported by Athena tables and views
	q := `SELECT column_name FROM information_schema.columns WHERE table_schema = '%s' AND table_name = '%s'`
	query := fmt.Sprintf(q, awsAthenaInfo.AthenaDatabase, awsAthenaInfo.AthenaTable)
	pageNum := 0
	athenaErr := aws.QueryAthenaPaginated(context.TODO(), query, func(page *athena.GetQueryResultsOutput) bool {
		if page == nil {
			log.Errorf("fetchColumns: Athena page is nil")
			return false
		} else if page.ResultSet == nil {
			log.Errorf("fetchColumns: Athena page.ResultSet is nil")
			return false
		}
		// remove header row 'column_name'
		rows := page.ResultSet.Rows[1:]
		for _, row := range rows {
			columnSet[*row.Data[0].VarCharValue] = true
		}
		pageNum++
		return true
	})

	if athenaErr != nil {
		return columnSet, athenaErr
	}

	if len(columnSet) == 0 {
		log.Infof("No columns retrieved from Athena")
	}

	return columnSet, nil
}

type spotInfo struct {
	Timestamp   string `csv:"Timestamp"`
	UsageType   string `csv:"UsageType"`
	Operation   string `csv:"Operation"`
	InstanceID  string `csv:"InstanceID"`
	MyBidID     string `csv:"MyBidID"`
	MyMaxPrice  string `csv:"MyMaxPrice"`
	MarketPrice string `csv:"MarketPrice"`
	Charge      string `csv:"Charge"`
	Version     string `csv:"Version"`
}

func (aws *AWS) parseSpotData(bucket string, prefix string, projectID string, region string) (map[string]*spotInfo, error) {

	aws.ConfigureAuth() // configure aws api authentication by setting env vars

	s3Prefix := projectID
	if len(prefix) != 0 {
		s3Prefix = prefix + "/" + s3Prefix
	}

	aak, err := aws.GetAWSAccessKey()
	if err != nil {
		return nil, err
	}

	cfg, err := aak.CreateConfig(region)
	if err != nil {
		return nil, err
	}
	cli := s3.NewFromConfig(cfg)
	downloader := manager.NewDownloader(cli)

	tNow := time.Now()
	tOneDayAgo := tNow.Add(time.Duration(-24) * time.Hour) // Also get files from one day ago to avoid boundary conditions
	ls := &s3.ListObjectsInput{
		Bucket: awsSDK.String(bucket),
		Prefix: awsSDK.String(s3Prefix + "." + tOneDayAgo.Format("2006-01-02")),
	}
	ls2 := &s3.ListObjectsInput{
		Bucket: awsSDK.String(bucket),
		Prefix: awsSDK.String(s3Prefix + "." + tNow.Format("2006-01-02")),
	}
	lso, err := cli.ListObjects(context.TODO(), ls)
	if err != nil {
		aws.serviceAccountChecks.set("bucketList", &ServiceAccountCheck{
			Message:        "Bucket List Permissions Available",
			Status:         false,
			AdditionalInfo: err.Error(),
		})
		return nil, err
	} else {
		aws.serviceAccountChecks.set("bucketList", &ServiceAccountCheck{
			Message: "Bucket List Permissions Available",
			Status:  true,
		})
	}
	lsoLen := len(lso.Contents)
	log.Debugf("Found %d spot data files from yesterday", lsoLen)
	if lsoLen == 0 {
		log.Debugf("ListObjects \"s3://%s/%s\" produced no keys", *ls.Bucket, *ls.Prefix)
	}
	lso2, err := cli.ListObjects(context.TODO(), ls2)
	if err != nil {
		return nil, err
	}
	lso2Len := len(lso2.Contents)
	log.Debugf("Found %d spot data files from today", lso2Len)
	if lso2Len == 0 {
		log.Debugf("ListObjects \"s3://%s/%s\" produced no keys", *ls2.Bucket, *ls2.Prefix)
	}

	// TODO: Worth it to use LastModifiedDate to determine if we should reparse the spot data?
	var keys []*string
	for _, obj := range lso.Contents {
		keys = append(keys, obj.Key)
	}
	for _, obj := range lso2.Contents {
		keys = append(keys, obj.Key)
	}

	header, err := csvutil.Header(spotInfo{}, "csv")
	if err != nil {
		return nil, err
	}
	fieldsPerRecord := len(header)

	spots := make(map[string]*spotInfo)
	for _, key := range keys {
		getObj := &s3.GetObjectInput{
			Bucket: awsSDK.String(bucket),
			Key:    key,
		}

		buf := manager.NewWriteAtBuffer([]byte{})
		_, err := downloader.Download(context.TODO(), buf, getObj)
		if err != nil {
			aws.serviceAccountChecks.set("objectList", &ServiceAccountCheck{
				Message:        "Object Get Permissions Available",
				Status:         false,
				AdditionalInfo: err.Error(),
			})
			return nil, err
		} else {
			aws.serviceAccountChecks.set("objectList", &ServiceAccountCheck{
				Message: "Object Get Permissions Available",
				Status:  true,
			})
		}

		r := bytes.NewReader(buf.Bytes())

		gr, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}

		csvReader := csv.NewReader(gr)
		csvReader.Comma = '\t'
		csvReader.FieldsPerRecord = fieldsPerRecord

		dec, err := csvutil.NewDecoder(csvReader, header...)
		if err != nil {
			return nil, err
		}

		var foundVersion string
		for {
			spot := spotInfo{}
			err := dec.Decode(&spot)
			csvParseErr, isCsvParseErr := err.(*csv.ParseError)
			if err == io.EOF {
				break
			} else if err == csvutil.ErrFieldCount || (isCsvParseErr && csvParseErr.Err == csv.ErrFieldCount) {
				rec := dec.Record()
				// the first two "Record()" will be the comment lines
				// and they show up as len() == 1
				// the first of which is "#Version"
				// the second of which is "#Fields: "
				if len(rec) != 1 {
					log.Infof("Expected %d spot info fields but received %d: %s", fieldsPerRecord, len(rec), rec)
					continue
				}
				if len(foundVersion) == 0 {
					spotFeedVersion := rec[0]
					log.Debugf("Spot feed version is \"%s\"", spotFeedVersion)
					matches := versionRx.FindStringSubmatch(spotFeedVersion)
					if matches != nil {
						foundVersion = matches[1]
						if foundVersion != supportedSpotFeedVersion {
							log.Infof("Unsupported spot info feed version: wanted \"%s\" got \"%s\"", supportedSpotFeedVersion, foundVersion)
							break
						}
					}
					continue
				} else if strings.Index(rec[0], "#") == 0 {
					continue
				} else {
					log.Infof("skipping non-TSV line: %s", rec)
					continue
				}
			} else if err != nil {
				log.Warnf("Error during spot info decode: %+v", err)
				continue
			}

			log.DedupedInfof(5, "Found spot info for: %s", spot.InstanceID)
			spots[spot.InstanceID] = &spot
		}
		gr.Close()
	}
	return spots, nil
}

// ApplyReservedInstancePricing TODO
func (aws *AWS) ApplyReservedInstancePricing(nodes map[string]*Node) {

}

func (aws *AWS) ServiceAccountStatus() *ServiceAccountStatus {
	return aws.serviceAccountChecks.getStatus()
}

func (aws *AWS) CombinedDiscountForNode(instanceType string, isPreemptible bool, defaultDiscount, negotiatedDiscount float64) float64 {
	return 1.0 - ((1.0 - defaultDiscount) * (1.0 - negotiatedDiscount))
}

// Regions returns a predefined list of AWS regions
func (aws *AWS) Regions() []string {
	return awsRegions
}
