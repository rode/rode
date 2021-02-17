package server

import (
	"context"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"go.uber.org/zap"
)

func (r *rodeServer) ListResources(ctx context.Context, request *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	log := r.logger.Named("ListResources")
	log.Debug("received request", zap.Any("ListResourcesRequest", request))

	occurrences, err := r.grafeasCommon.ListOccurrences(ctx, &grafeas_proto.ListOccurrencesRequest{
		Parent:    "projects/rode",
		Filter:    request.Filter,
		PageSize:  request.PageSize,
		PageToken: request.PageToken,
	})
	if err != nil {
		log.Error("failed to list occurrences", zap.Error(err))
		return nil, err
	}

	uniqueResources := make(map[string]string)
	for _, occurrence := range occurrences.Occurrences {
		if _, ok := uniqueResources[occurrence.Resource.Uri]; ok {
			continue
		}

		uniqueResources[occurrence.Resource.Uri] = occurrence.Resource.Uri
	}

	resources := make([]*grafeas_proto.Resource, 0)
	for resourceUri := range uniqueResources {
		resources = append(resources, &grafeas_proto.Resource{
			Uri: resourceUri,
		})
	}

	return &pb.ListResourcesResponse{
		Resources: resources,
		// NextPageToken for the next set of occurrences -- this doesn't index into how many resources
		// there are
		NextPageToken: occurrences.NextPageToken,
	}, nil
}
