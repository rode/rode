package grafeas

import (
	"context"
	"fmt"
	"github.com/rode/rode/pkg/constants"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"go.uber.org/zap"
	"strings"
)

//go:generate counterfeiter -generate

//counterfeiter:generate . Helper
type Helper interface {
	ListVersionedResourceOccurrences(ctx context.Context, resourceUri, pageToken string, pageSize int32) ([]*grafeas_proto.Occurrence, string, error)
}

type helper struct {
	logger        *zap.Logger
	grafeasCommon grafeas_proto.GrafeasV1Beta1Client
}

func NewHelper(logger *zap.Logger, grafeasCommon grafeas_proto.GrafeasV1Beta1Client) Helper {
	return &helper{logger, grafeasCommon}
}

func (h *helper) ListVersionedResourceOccurrences(ctx context.Context, resourceUri, pageToken string, pageSize int32) ([]*grafeas_proto.Occurrence, string, error) {
	log := h.logger.Named("ListVersionedResourceOccurrences")

	buildOccurrences, err := h.grafeasCommon.ListOccurrences(ctx, &grafeas_proto.ListOccurrencesRequest{
		Parent:   constants.RodeProjectSlug,
		PageSize: constants.MaxPageSize,
		Filter:   fmt.Sprintf(`kind == "BUILD" && (resource.uri == "%[1]s" || build.provenance.builtArtifacts.nestedFilter(id == "%[1]s"))`, resourceUri),
	})
	if err != nil {
		return nil, "", fmt.Errorf("error fetching build occurrences: %v", err)
	}

	resourceUris := map[string]string{
		resourceUri: resourceUri,
	}
	for _, occurrence := range buildOccurrences.Occurrences {
		resourceUris[occurrence.Resource.Uri] = occurrence.Resource.Uri
		for _, artifact := range occurrence.GetBuild().GetProvenance().BuiltArtifacts {
			resourceUris[artifact.Id] = artifact.Id
		}
	}

	var resourceFilters []string
	for k := range resourceUris {
		resourceFilters = append(resourceFilters, fmt.Sprintf(`resource.uri == "%s"`, k))
	}

	filter := strings.Join(resourceFilters, " || ")
	log.Debug("listing occurrences", zap.String("filter", filter))
	allOccurrences, err := h.grafeasCommon.ListOccurrences(ctx, &grafeas_proto.ListOccurrencesRequest{
		Parent:    constants.RodeProjectSlug,
		Filter:    filter,
		PageSize:  pageSize,
		PageToken: pageToken,
	})
	if err != nil {
		return nil, "", fmt.Errorf("error listing all occurrences: %v", err)
	}

	return allOccurrences.Occurrences, allOccurrences.NextPageToken, nil
}
