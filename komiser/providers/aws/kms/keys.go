package kms

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	. "github.com/mlabouardy/komiser/models"
	. "github.com/mlabouardy/komiser/providers"
)

func Keys(ctx context.Context, client ProviderClient) ([]Resource, error) {
	var config kms.ListKeysInput
	resources := make([]Resource, 0)
	kmsClient := kms.NewFromConfig(*client.AWSClient)

	for {
		output, err := kmsClient.ListKeys(ctx, &config)
		if err != nil {
			return resources, err
		}

		for _, key := range output.Keys {
			outputTags, err := kmsClient.ListResourceTags(ctx, &kms.ListResourceTagsInput{
				KeyId: key.KeyId,
			})

			tags := make([]Tag, 0)

			if err == nil {
				for _, tag := range outputTags.Tags {
					tags = append(tags, Tag{
						Key:   *tag.TagKey,
						Value: *tag.TagValue,
					})
				}
			}

			resources = append(resources, Resource{
				Provider:   "AWS",
				Account:    client.Name,
				Service:    "KMS",
				Region:     client.AWSClient.Region,
				ResourceId: *key.KeyArn,
				Cost:       0,
				Name:       *key.KeyId,
				FetchedAt:  time.Now(),
				Tags:       tags,
				Link:       fmt.Sprintf("https:/%s.console.aws.amazon.com/kms/home?region=%s#/kms/keys/%s", client.AWSClient.Region, client.AWSClient.Region, *key.KeyId),
			})
		}

		if aws.ToString(output.NextMarker) == "" {
			break
		}

		config.Marker = output.NextMarker
	}
	log.WithFields(log.Fields{
		"provider":  "AWS",
		"account":   client.Name,
		"region":    client.AWSClient.Region,
		"service":   "KMS",
		"resources": len(resources),
	}).Info("Fetched resources")
	return resources, nil
}
