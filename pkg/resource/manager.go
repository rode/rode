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
	"github.com/rode/es-index-manager/indexmanager"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
	"github.com/rode/rode/config"
	"github.com/rode/rode/pkg/constants"
	"github.com/rode/rode/pkg/util"
	pb "github.com/rode/rode/proto/v1alpha1"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate counterfeiter -generate

const (
	resourcesDocumentKind = "resources"

	resourceDocumentJoinField   = "join"
	resourceRelationName        = "resource"
	resourceVersionRelationName = "version"
)

//counterfeiter:generate . Manager
type Manager interface {
	BatchCreateResources(ctx context.Context, occurrences []*grafeas_proto.Occurrence) error
	BatchCreateResourceVersions(ctx context.Context, occurrences []*grafeas_proto.Occurrence) error
	ListResources(ctx context.Context, request *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error)
	ListResourceVersions(ctx context.Context, request *pb.ListResourceVersionsRequest) (*pb.ListResourceVersionsResponse, error)
	GetResource(ctx context.Context, resourceId string) (*pb.Resource, error)
	GetResourceVersion(ctx context.Context, resourceUri string) (*pb.ResourceVersion, error)
}

type manager struct {
	logger       *zap.Logger
	esClient     esutil.Client
	esConfig     *config.ElasticsearchConfig
	indexManager indexmanager.IndexManager
	filterer     filtering.Filterer
}

func NewManager(logger *zap.Logger, esClient esutil.Client, esConfig *config.ElasticsearchConfig, indexManager indexmanager.IndexManager, filterer filtering.Filterer) Manager {
	return &manager{
		logger:       logger,
		esClient:     esClient,
		esConfig:     esConfig,
		indexManager: indexManager,
		filterer:     filterer,
	}
}

// BatchCreateResources creates resources from a list of occurrences. This method is intended to be invoked
// as a side effect of BatchCreateOccurrences.
func (m *manager) BatchCreateResources(ctx context.Context, occurrences []*grafeas_proto.Occurrence) error {
	log := m.logger.Named("BatchCreateResources")

	resources := map[string]*pb.Resource{}
	var resourceIds []string
	for _, occurrence := range occurrences {
		uriParts, err := parseResourceUri(occurrence.Resource.Uri)
		if err != nil {
			return err
		}

		resource := &pb.Resource{
			Name:    uriParts.name,
			Type:    uriParts.resourceType,
			Created: occurrence.CreateTime,
		}
		resourceId := uriParts.prefixedName

		if _, ok := resources[resourceId]; ok {
			continue
		}
		resources[resourceId] = resource
		resourceIds = append(resourceIds, resourceId)
	}

	multiGetResponse, err := m.esClient.MultiGet(ctx, &esutil.MultiGetRequest{
		DocumentIds: resourceIds,
		Index:       m.indexManager.AliasName(resourcesDocumentKind, ""),
	})
	if err != nil {
		return err
	}

	var bulkItems []*esutil.BulkRequestItem
	for i, resourceId := range resourceIds {
		if multiGetResponse.Docs[i].Found {
			log.Debug("skipping resource creation because it already exists", zap.String("resourceId", resourceId))
			continue
		}

		log.Debug("Adding resource to bulk request", zap.String("resourceId", resourceId))

		bulkItems = append(bulkItems, &esutil.BulkRequestItem{
			Operation:  esutil.BULK_INDEX,
			Message:    resources[resourceId],
			DocumentId: resourceId,
			Join: &esutil.EsJoin{
				Field: resourceDocumentJoinField,
				Name:  resourceRelationName,
			},
		})
	}

	if len(bulkItems) == 0 {
		return nil
	}

	bulkResponse, err := m.esClient.Bulk(ctx, &esutil.BulkRequest{
		Index:   m.indexManager.AliasName(resourcesDocumentKind, ""),
		Refresh: m.esConfig.Refresh.String(),
		Items:   bulkItems,
	})
	if err != nil {
		return err
	}

	var bulkItemErrors []error
	for _, item := range bulkResponse.Items {
		if item.Index.Error != nil {
			bulkItemErrors = append(bulkItemErrors, fmt.Errorf("error creating resource [%d] %s: %s", item.Index.Status, item.Index.Error.Type, item.Index.Error.Reason))
		}
	}

	if len(bulkItemErrors) > 0 {
		return fmt.Errorf("failed to create all resources: %v", bulkItemErrors)
	}

	return nil
}

// BatchCreateResourceVersions creates resource versions from a list of occurrences. This method is intended
// to be invoked as a side effect of BatchCreateOccurrences, after BatchCreateResources
func (m *manager) BatchCreateResourceVersions(ctx context.Context, occurrences []*grafeas_proto.Occurrence) error {
	log := m.logger.Named("BatchCreateResourceVersions")

	resourceVersions := map[string]*pb.ResourceVersion{}
	var versionIds []string

	// build a list of resource versions from occurrences. these may or may not already exist
	for _, occurrence := range occurrences {
		newVersions := resourceVersionsFromOccurrence(occurrence)

		// check if we already know about these versions
		for versionId, newVersion := range newVersions {
			if existingVersion, ok := resourceVersions[versionId]; ok {
				// if we do, update the names and create time, if needed
				if newVersion.Created != nil {
					existingVersion.Created = newVersion.Created
				}

				if newVersion.Names != nil {
					existingVersion.Names = newVersion.Names
				}
			} else {
				resourceVersions[versionId] = newVersion
				versionIds = append(versionIds, versionId)
			}
		}
	}

	// check which resource versions already exist
	multiGetResponse, err := m.esClient.MultiGet(ctx, &esutil.MultiGetRequest{
		DocumentIds: versionIds,
		Index:       m.indexManager.AliasName(resourcesDocumentKind, ""),
	})
	if err != nil {
		return err
	}

	// versions that don't exist need to be created
	// versions that do exist may need to be updated
	var bulkItems []*esutil.BulkRequestItem
	for i, versionId := range versionIds {
		version := resourceVersions[versionId]
		uriParts, err := parseResourceUri(versionId)
		if err != nil {
			return err
		}

		bulkItem := &esutil.BulkRequestItem{
			Operation:  esutil.BULK_INDEX,
			Message:    version,
			DocumentId: versionId,
			Join: &esutil.EsJoin{
				Field:  resourceDocumentJoinField,
				Name:   resourceVersionRelationName,
				Parent: uriParts.prefixedName,
			},
		}

		if multiGetResponse.Docs[i].Found {
			// if the version already exists, we may have to update(index) it.
			// if we have a list of names, update the existing version with the new one
			// otherwise, we have nothing to do here
			if len(version.Names) != 0 {
				log.Debug("updating resource version", zap.Any("version", version))
			} else {
				bulkItem = nil
			}
		} else {
			// if the version doesn't exist, we need to create it
			// before creating the version, add a timestamp if it doesn't exist
			if version.Created == nil {
				version.Created = timestamppb.Now()
			}

			log.Debug("creating resource version", zap.Any("version", version))
		}

		if bulkItem != nil {
			bulkItems = append(bulkItems, bulkItem)
		}
	}

	// perform bulk create / update
	var bulkItemErrors []error
	if len(bulkItems) != 0 {
		bulkResponse, err := m.esClient.Bulk(ctx, &esutil.BulkRequest{
			Index:   m.indexManager.AliasName(resourcesDocumentKind, ""),
			Refresh: m.esConfig.Refresh.String(),
			Items:   bulkItems,
		})
		if err != nil {
			return err
		}

		for _, item := range bulkResponse.Items {
			if item.Index.Error != nil {
				bulkItemErrors = append(bulkItemErrors, fmt.Errorf("error indexing resource [%d] %s: %s", item.Index.Status, item.Index.Error.Type, item.Index.Error.Reason))
			}
		}
	}

	if len(bulkItemErrors) > 0 {
		return fmt.Errorf("failed to create/update some resource versions: %v", bulkItemErrors)
	}

	return nil
}

func (m *manager) ListResources(ctx context.Context, request *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	// resources and their versions are stored in the same index. we need to specify the join field
	// in this query in order to only select resources. we use a "must" here so we can combine this query with
	// the one generated by the filter, if provided
	queries := filtering.Must{
		&filtering.Query{
			Term: &filtering.Term{
				resourceDocumentJoinField: resourceRelationName,
			},
		},
	}

	if request.Filter != "" {
		filterQuery, err := m.filterer.ParseExpression(request.Filter)
		if err != nil {
			return nil, err
		}

		queries = append(queries, filterQuery)
	}

	searchRequest := &esutil.SearchRequest{
		Index: m.indexManager.AliasName(resourcesDocumentKind, ""),
		Search: &esutil.EsSearch{
			Query: &filtering.Query{
				Bool: &filtering.Bool{
					Must: &queries,
				},
			},
		},
	}

	if request.PageSize != 0 {
		searchRequest.Pagination = &esutil.SearchPaginationOptions{
			Size:  int(request.PageSize),
			Token: request.PageToken,
		}
	}

	searchResponse, err := m.esClient.Search(ctx, searchRequest)
	if err != nil {
		return nil, err
	}

	var resources []*pb.Resource
	for _, hit := range searchResponse.Hits.Hits {
		var resource pb.Resource
		err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(hit.Source, &resource)
		if err != nil {
			return nil, err
		}

		resource.Id = hit.ID
		resources = append(resources, &resource)
	}

	return &pb.ListResourcesResponse{
		Resources:     resources,
		NextPageToken: searchResponse.NextPageToken,
	}, nil
}

// ListResourceVersions handles the main logic for fetching versions associated with a resource.
func (m *manager) ListResourceVersions(ctx context.Context, request *pb.ListResourceVersionsRequest) (*pb.ListResourceVersionsResponse, error) {
	// resources and their versions are stored in the same index. we need to specify the "has_parent" field
	// in this query in order to only select resource versions with the provided resource id as the parent.
	// we use a "must" here so we can combine this query with the one generated by the filter, if provided
	queries := filtering.Must{
		&filtering.Query{
			HasParent: &filtering.HasParent{
				ParentType: resourceRelationName,
				Query: &filtering.Query{
					Term: &filtering.Term{
						"_id": request.Id,
					},
				},
			},
		},
	}

	if request.Filter != "" {
		filterQuery, err := m.filterer.ParseExpression(request.Filter)
		if err != nil {
			return nil, err
		}

		queries = append(queries, filterQuery)
	}

	searchRequest := &esutil.SearchRequest{
		Index: m.indexManager.AliasName(resourcesDocumentKind, ""),
		Search: &esutil.EsSearch{
			Query: &filtering.Query{
				Bool: &filtering.Bool{
					Must: &queries,
				},
			},
			Sort: map[string]esutil.EsSortOrder{
				"created": esutil.EsSortOrderDescending,
			},
		},
	}

	if request.PageSize != 0 {
		searchRequest.Pagination = &esutil.SearchPaginationOptions{
			Size:  int(request.PageSize),
			Token: request.PageToken,
		}
	}

	searchResponse, err := m.esClient.Search(ctx, searchRequest)
	if err != nil {
		return nil, err
	}

	var resourceVersions []*pb.ResourceVersion
	for _, hit := range searchResponse.Hits.Hits {
		var resourceVersion pb.ResourceVersion
		err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(hit.Source, &resourceVersion)
		if err != nil {
			return nil, err
		}

		resourceVersions = append(resourceVersions, &resourceVersion)
	}

	return &pb.ListResourceVersionsResponse{
		Versions:      resourceVersions,
		NextPageToken: searchResponse.NextPageToken,
	}, nil
}

func (m *manager) GetResource(ctx context.Context, resourceId string) (*pb.Resource, error) {
	log := m.logger.Named("GetResource").With(zap.String("resource", resourceId))

	response, err := m.esClient.Get(ctx, &esutil.GetRequest{
		Index:      m.indexManager.AliasName(resourcesDocumentKind, ""),
		DocumentId: resourceId,
	})
	if err != nil {
		return nil, err
	}

	if !response.Found {
		log.Debug("resource not found")

		return nil, nil
	}

	var resource pb.Resource
	err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(response.Source, &resource)
	if err != nil {
		return nil, err
	}

	return &resource, nil
}

func (m *manager) GetResourceVersion(ctx context.Context, resourceUri string) (*pb.ResourceVersion, error) {
	log := m.logger.Named("GetResourceVersion")
	uriParts, err := parseResourceUri(resourceUri)
	if err != nil {
		return nil, util.GrpcInternalError(log, "error parsing resource uri", err)
	}

	// the id of the version is the full resource uri, and the parent (for routing) is the prefixed name
	response, err := m.esClient.Get(ctx, &esutil.GetRequest{
		Index:      m.indexManager.AliasName(constants.ResourcesDocumentKind, ""),
		DocumentId: resourceUri,
		Routing:    uriParts.prefixedName,
	})
	if err != nil {
		return nil, util.GrpcInternalError(log, "error fetching resource for evaluation", err)
	}
	if !response.Found {
		return nil, util.GrpcErrorWithCode(log, fmt.Sprintf("resource with uri %s not found", resourceUri), nil, codes.NotFound)
	}

	var resourceVersion pb.ResourceVersion
	err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(response.Source, &resourceVersion)
	if err != nil {
		return nil, err
	}

	return &resourceVersion, nil
}

// resourceVersionsFromOccurrence will create a map of versions, keyed by their IDs, from an occurrence.
// if the occurrence is a build occurrence, resource versions will also be created for each built artifact.
func resourceVersionsFromOccurrence(o *grafeas_proto.Occurrence) map[string]*pb.ResourceVersion {
	result := make(map[string]*pb.ResourceVersion)

	result[o.Resource.Uri] = &pb.ResourceVersion{
		Version: o.Resource.Uri,
	}

	if o.Kind == common_go_proto.NoteKind_BUILD {
		details := o.Details.(*grafeas_proto.Occurrence_Build)

		for _, artifact := range details.Build.Provenance.BuiltArtifacts {
			result[artifact.Id] = &pb.ResourceVersion{
				Version: artifact.Id,
				Names:   artifact.Names,
				Created: o.CreateTime,
			}
		}
	}

	return result
}
