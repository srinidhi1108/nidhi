package storage

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mlabouardy/komiser/models"
	"github.com/mlabouardy/komiser/providers"
)

func Volumes(ctx context.Context, client providers.ProviderClient) ([]models.Resource, error) {
	resources := make([]models.Resource, 0)

	volumes, err := client.CivoClient.ListVolumes()
	if err != nil {
		return resources, err
	}

	for _, volume := range volumes {

		resources = append(resources, models.Resource{
			Provider:   "Civo",
			Account:    client.Name,
			Service:    "Volume",
			Region:     client.CivoClient.Region,
			ResourceId: volume.ID,
			Cost:       0,
			Name:       volume.Name,
			FetchedAt:  time.Now(),
			CreatedAt:  volume.CreatedAt,
			Link:       "https://dashboard.civo.com/volumes",
		})
	}

	log.WithFields(log.Fields{
		"provider":  "Civo",
		"account":   client.Name,
		"service":   "Volume",
		"region":    client.CivoClient.Region,
		"resources": len(resources),
	}).Info("Fetched resources")
	return resources, nil
}
