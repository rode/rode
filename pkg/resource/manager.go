// Copyright 2021 The Rode Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rode/es-index-manager/indexmanager"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/rode/config"
	pb "github.com/rode/rode/proto/v1alpha1"
	"go.uber.org/zap"
)

//go:generate counterfeiter -generate

const (
	genericResourcesDocumentKind = "generic-resources"
)

//counterfeiter:generate . Manager
type Manager interface {
	BatchCreateGenericResources(context.Context, *pb.BatchCreateOccurrencesRequest) error
}

type manager struct {
	logger       *zap.Logger
	esClient     esutil.Client
	esConfig     *config.ElasticsearchConfig
	indexManager indexmanager.IndexManager
}

func NewManager(logger *zap.Logger, esClient esutil.Client, esConfig *config.ElasticsearchConfig, indexManager indexmanager.IndexManager) Manager {
	return &manager{
		logger:       logger,
		esClient:     esClient,
		esConfig:     esConfig,
		indexManager: indexManager,
	}
}

// BatchCreateGenericResources creates generic resources from a list of occurrences. This method is intended to be invoked
// as a side effect of BatchCreateOccurrences.
func (m *manager) BatchCreateGenericResources(ctx context.Context, request *pb.BatchCreateOccurrencesRequest) error {
	log := m.logger.Named("BatchCreateGenericResources")

	genericResources := map[string]*pb.GenericResource{}
	var resourceIds []string
	for _, x := range request.Occurrences {
		uriParts, err := parseResourceUri(x.Resource.Uri)
		if err != nil {
			return err
		}

		genericResource := &pb.GenericResource{
			Name: uriParts.name,
			Type: uriParts.resourceType,
		}
		resourceId := genericResourceId(genericResource)

		if _, ok := genericResources[resourceId]; ok {
			continue
		}
		genericResources[resourceId] = genericResource
		resourceIds = append(resourceIds, resourceId)
	}

	multiGetResponse, err := m.esClient.MultiGet(ctx, &esutil.MultiGetRequest{
		DocumentIds: resourceIds,
		Index:       m.indexManager.AliasName(genericResourcesDocumentKind, ""),
	})
	if err != nil {
		return err
	}

	var bulkCreateItems []*esutil.BulkCreateRequestItem
	for i, resourceId := range resourceIds {
		if multiGetResponse.Docs[i].Found {
			log.Debug("skipping resource creation because it already exists", zap.String("resourceId", resourceId))
			continue
		}

		log.Debug("Adding resource to bulk request", zap.String("resourceId", resourceId))

		bulkCreateItems = append(bulkCreateItems, &esutil.BulkCreateRequestItem{
			Message:    genericResources[resourceId],
			DocumentId: resourceId,
		})
	}

	if len(bulkCreateItems) == 0 {
		return nil
	}

	bulkCreateResponse, err := m.esClient.BulkCreate(ctx, &esutil.BulkCreateRequest{
		Index:   m.indexManager.AliasName(genericResourcesDocumentKind, ""),
		Refresh: m.esConfig.Refresh.String(),
		Items:   bulkCreateItems,
	})
	if err != nil {
		return err
	}

	var bulkItemErrors []error
	for i := range bulkCreateResponse.Items {
		item := bulkCreateResponse.Items[i].Create
		if item.Error != nil && item.Status != http.StatusConflict {
			itemError := fmt.Errorf("error creating generic resource [%d] %s: %s", item.Status, item.Error.Type, item.Error.Reason)
			bulkItemErrors = append(bulkItemErrors, itemError)
		}
	}

	if len(bulkItemErrors) > 0 {
		return fmt.Errorf("failed to create all resources: %v", bulkItemErrors)
	}

	return nil
}

func genericResourceId(r *pb.GenericResource) string {
	return fmt.Sprintf("%s:%s", r.Type, r.Name)
}
