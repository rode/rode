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

package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rode/rode/pkg/constants"
	"github.com/rode/rode/pkg/grafeas"

	"github.com/rode/rode/pkg/policy"
	"github.com/rode/rode/pkg/resource"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"

	"github.com/rode/es-index-manager/indexmanager"

	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	grafeas_project_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/project_go_proto"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewRodeServer constructor for rodeServer
func NewRodeServer(
	logger *zap.Logger,
	grafeasCommon grafeas_proto.GrafeasV1Beta1Client,
	grafeasProjects grafeas_project_proto.ProjectsClient,
	grafeasExtensions grafeas.Extensions,
	resourceManager resource.Manager,
	indexManager indexmanager.IndexManager,
	policyManager policy.Manager,
	policyGroupManager policy.PolicyGroupManager,
) (pb.RodeServer, error) {
	rodeServer := &rodeServer{
		logger,
		grafeasCommon,
		grafeasProjects,
		grafeasExtensions,
		resourceManager,
		indexManager,
		policyManager,
		policyGroupManager,
	}

	if err := rodeServer.initialize(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize rode server: %s", err)
	}

	return rodeServer, nil
}

type rodeServer struct {
	logger            *zap.Logger
	grafeasCommon     grafeas_proto.GrafeasV1Beta1Client
	grafeasProjects   grafeas_project_proto.ProjectsClient
	grafeasExtensions grafeas.Extensions
	resourceManager   resource.Manager
	indexManager      indexmanager.IndexManager
	policy.Manager
	policy.PolicyGroupManager
}

func (r *rodeServer) BatchCreateOccurrences(ctx context.Context, occurrenceRequest *pb.BatchCreateOccurrencesRequest) (*pb.BatchCreateOccurrencesResponse, error) {
	log := r.logger.Named("BatchCreateOccurrences")
	log.Debug("received request", zap.Any("BatchCreateOccurrencesRequest", occurrenceRequest))

	occurrenceResponse, err := r.grafeasCommon.BatchCreateOccurrences(ctx, &grafeas_proto.BatchCreateOccurrencesRequest{
		Parent:      constants.RodeProjectSlug,
		Occurrences: occurrenceRequest.GetOccurrences(),
	})
	if err != nil {
		return nil, createError(log, "error creating occurrences", err)
	}

	if err = r.resourceManager.BatchCreateResources(ctx, occurrenceResponse.Occurrences); err != nil {
		return nil, createError(log, "error creating resources", err)
	}

	if err = r.resourceManager.BatchCreateResourceVersions(ctx, occurrenceResponse.Occurrences); err != nil {
		return nil, createError(log, "error creating resource versions", err)
	}

	return &pb.BatchCreateOccurrencesResponse{
		Occurrences: occurrenceResponse.GetOccurrences(),
	}, nil
}

func (r *rodeServer) ListResources(ctx context.Context, request *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	log := r.logger.Named("ListResources")
	log.Debug("received request", zap.Any("request", request))

	return r.resourceManager.ListResources(ctx, request)
}

func (r *rodeServer) ListResourceVersions(ctx context.Context, request *pb.ListResourceVersionsRequest) (*pb.ListResourceVersionsResponse, error) {
	log := r.logger.Named("ListResourceVersions").With(zap.Any("resource", request.Id))

	if request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "resource id is required")
	}

	resource, err := r.resourceManager.GetResource(ctx, request.Id)
	if err != nil {
		return nil, err
	}

	if resource == nil {
		log.Debug("resource not found")

		return nil, status.Error(codes.NotFound, fmt.Sprintf("resource with id %s not found", request.Id))
	}

	return r.resourceManager.ListResourceVersions(ctx, request)
}

func (r *rodeServer) initialize(ctx context.Context) error {
	log := r.logger.Named("initialize")

	if err := r.indexManager.Initialize(ctx); err != nil {
		return fmt.Errorf("error initializing index manager: %s", err)
	}

	_, err := r.grafeasProjects.GetProject(ctx, &grafeas_project_proto.GetProjectRequest{Name: constants.RodeProjectSlug})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			_, err := r.grafeasProjects.CreateProject(ctx, &grafeas_project_proto.CreateProjectRequest{Project: &grafeas_project_proto.Project{Name: constants.RodeProjectSlug}})
			if err != nil {
				log.Error("failed to create rode project", zap.Error(err))
				return err
			}
			log.Info("created rode project")
		} else {
			log.Error("error checking if rode project exists", zap.Error(err))
			return err
		}
	}

	indexSettings := []struct {
		indexName    string
		aliasName    string
		documentKind string
	}{
		{
			indexName:    r.indexManager.IndexName(constants.PoliciesDocumentKind, ""),
			aliasName:    r.indexManager.AliasName(constants.PoliciesDocumentKind, ""),
			documentKind: constants.PoliciesDocumentKind,
		},
		{
			indexName:    r.indexManager.IndexName(constants.ResourcesDocumentKind, ""),
			aliasName:    r.indexManager.AliasName(constants.ResourcesDocumentKind, ""),
			documentKind: constants.ResourcesDocumentKind,
		},
		{
			indexName:    r.indexManager.IndexName(constants.PolicyGroupsDocumentKind, ""),
			aliasName:    r.indexManager.AliasName(constants.PolicyGroupsDocumentKind, ""),
			documentKind: constants.PolicyGroupsDocumentKind,
		},
	}

	for _, settings := range indexSettings {
		if err := r.indexManager.CreateIndex(ctx, settings.indexName, settings.aliasName, settings.documentKind); err != nil {
			return fmt.Errorf("error creating index: %s", err)
		}
	}

	return nil
}

func (r *rodeServer) ListVersionedResourceOccurrences(ctx context.Context, request *pb.ListVersionedResourceOccurrencesRequest) (*pb.ListVersionedResourceOccurrencesResponse, error) {
	log := r.logger.Named("ListVersionedResourceOccurrences")
	log.Debug("received request", zap.Any("ListVersionedResourceOccurrencesRequest", request))

	resourceUri := request.ResourceUri
	if resourceUri == "" {
		return nil, createErrorWithCode(log, "invalid request", errors.New("must set resource_uri"), codes.InvalidArgument)
	}

	occurrences, nextPageToken, err := r.grafeasExtensions.ListVersionedResourceOccurrences(ctx, resourceUri, request.PageToken, request.PageSize)
	if err != nil {
		return nil, createError(log, "error listing versioned resource occurrences", err)
	}

	response := &pb.ListVersionedResourceOccurrencesResponse{
		Occurrences:   occurrences,
		NextPageToken: nextPageToken,
	}

	if request.FetchRelatedNotes {
		relatedNotes, err := r.fetchRelatedNotes(ctx, log, occurrences)
		if err != nil {
			return nil, createError(log, "error fetching related notes", err)
		}

		response.RelatedNotes = relatedNotes
	}

	return response, nil
}

func (r *rodeServer) fetchRelatedNotes(ctx context.Context, logger *zap.Logger, occurrences []*grafeas_proto.Occurrence) (map[string]*grafeas_proto.Note, error) {
	log := logger.Named("fetchRelatedNotes")

	if len(occurrences) == 0 {
		return nil, nil
	}

	noteFiltersMap := make(map[string]string)
	for _, occurrence := range occurrences {
		if _, ok := noteFiltersMap[occurrence.NoteName]; !ok {
			noteFiltersMap[occurrence.NoteName] = fmt.Sprintf(`"name" == "%s"`, occurrence.NoteName)
		}
	}

	var noteFilters []string
	for _, filter := range noteFiltersMap {
		noteFilters = append(noteFilters, filter)
	}

	log.Debug("fetching related notes")
	listNotesResponse, err := r.grafeasCommon.ListNotes(ctx, &grafeas_proto.ListNotesRequest{
		Parent:   constants.RodeProjectSlug,
		Filter:   strings.Join(noteFilters, " || "),
		PageSize: constants.MaxPageSize,
	})
	if err != nil {
		return nil, err
	}

	result := make(map[string]*grafeas_proto.Note)
	for _, note := range listNotesResponse.Notes {
		result[note.Name] = note
	}

	return result, nil
}

func (r *rodeServer) ListOccurrences(ctx context.Context, occurrenceRequest *pb.ListOccurrencesRequest) (*pb.ListOccurrencesResponse, error) {
	log := r.logger.Named("ListOccurrences")
	log.Debug("received request", zap.Any("ListOccurrencesRequest", occurrenceRequest))

	request := &grafeas_proto.ListOccurrencesRequest{
		Parent:    constants.RodeProjectSlug,
		Filter:    occurrenceRequest.Filter,
		PageToken: occurrenceRequest.PageToken,
		PageSize:  occurrenceRequest.PageSize,
	}

	listOccurrencesResponse, err := r.grafeasCommon.ListOccurrences(ctx, request)
	if err != nil {
		return nil, createError(log, "error listing occurrences", err)
	}

	return &pb.ListOccurrencesResponse{
		Occurrences:   listOccurrencesResponse.GetOccurrences(),
		NextPageToken: listOccurrencesResponse.GetNextPageToken(),
	}, nil
}

func (r *rodeServer) UpdateOccurrence(ctx context.Context, occurrenceRequest *pb.UpdateOccurrenceRequest) (*grafeas_proto.Occurrence, error) {
	log := r.logger.Named("UpdateOccurrence")
	log.Debug("received request", zap.Any("UpdateOccurrenceRequest", occurrenceRequest))

	name := fmt.Sprintf("projects/rode/occurrences/%s", occurrenceRequest.Id)

	if occurrenceRequest.Occurrence.Name != name {
		log.Error("occurrence name does not contain the occurrence id", zap.String("occurrenceName", occurrenceRequest.Occurrence.Name), zap.String("id", occurrenceRequest.Id))
		return nil, status.Error(codes.InvalidArgument, "occurrence name does not contain the occurrence id")
	}

	updatedOccurrence, err := r.grafeasCommon.UpdateOccurrence(ctx, &grafeas_proto.UpdateOccurrenceRequest{
		Name:       name,
		Occurrence: occurrenceRequest.Occurrence,
		UpdateMask: occurrenceRequest.UpdateMask,
	})
	if err != nil {
		return nil, createError(log, "error updating occurrence", err)
	}

	return updatedOccurrence, nil
}

func (r *rodeServer) RegisterCollector(ctx context.Context, registerCollectorRequest *pb.RegisterCollectorRequest) (*pb.RegisterCollectorResponse, error) {
	log := r.logger.Named("RegisterCollector")

	if registerCollectorRequest.Id == "" {
		return nil, createErrorWithCode(log, "collector ID is required", nil, codes.InvalidArgument)
	}

	if len(registerCollectorRequest.Notes) == 0 {
		return &pb.RegisterCollectorResponse{}, nil
	}

	// build collection of notes that potentially need to be created
	notesWithIds := make(map[string]*grafeas_proto.Note)
	notesToCreate := make(map[string]*grafeas_proto.Note)
	for _, note := range registerCollectorRequest.Notes {
		noteId := buildNoteIdFromCollectorId(registerCollectorRequest.Id, note)

		if _, ok := notesWithIds[noteId]; ok {
			return nil, createErrorWithCode(log, "cannot use more than one note type when registering a collector", nil, codes.InvalidArgument)
		}

		notesWithIds[noteId] = note
		notesToCreate[noteId] = note
	}

	log = log.With(zap.Any("notes", notesWithIds))

	// find out which notes already exist
	filter := fmt.Sprintf(`name.startsWith("%s/notes/%s-")`, constants.RodeProjectSlug, registerCollectorRequest.Id)
	listNotesResponse, err := r.grafeasCommon.ListNotes(ctx, &grafeas_proto.ListNotesRequest{
		Parent: constants.RodeProjectSlug,
		Filter: filter,
	})
	if err != nil {
		return nil, createError(log, "error listing notes", err)
	}

	// build map of notes that need to be created
	for _, note := range listNotesResponse.Notes {
		noteId := getNoteIdFromNoteName(note.Name)

		if _, ok := notesWithIds[noteId]; ok {
			notesWithIds[noteId].Name = note.Name
			delete(notesToCreate, noteId)
		}
	}

	if len(notesToCreate) != 0 {
		batchCreateNotesResponse, err := r.grafeasCommon.BatchCreateNotes(ctx, &grafeas_proto.BatchCreateNotesRequest{
			Parent: constants.RodeProjectSlug,
			Notes:  notesToCreate,
		})
		if err != nil {
			return nil, createError(log, "error creating notes", err)
		}

		for _, note := range batchCreateNotesResponse.Notes {
			noteId := getNoteIdFromNoteName(note.Name)

			if _, ok := notesWithIds[noteId]; ok {
				notesWithIds[noteId].Name = note.Name
			}
		}
	}

	return &pb.RegisterCollectorResponse{
		Notes: notesWithIds,
	}, nil
}

// CreateNote operates as a simple proxy to grafeas.CreateNote, for now.
func (r *rodeServer) CreateNote(ctx context.Context, request *pb.CreateNoteRequest) (*grafeas_proto.Note, error) {
	log := r.logger.Named("CreateNote").With(zap.String("noteId", request.NoteId))

	log.Debug("creating note in grafeas")

	return r.grafeasCommon.CreateNote(ctx, &grafeas_proto.CreateNoteRequest{
		Parent: constants.RodeProjectSlug,
		NoteId: request.NoteId,
		Note:   request.Note,
	})
}

// createError is a helper function that allows you to easily log an error and return a gRPC formatted error.
func createError(log *zap.Logger, message string, err error, fields ...zap.Field) error {
	return createErrorWithCode(log, message, err, codes.Internal, fields...)
}

// createError is a helper function that allows you to easily log an error and return a gRPC formatted error.
func createErrorWithCode(log *zap.Logger, message string, err error, code codes.Code, fields ...zap.Field) error {
	if err == nil {
		log.Error(message, fields...)
		return status.Errorf(code, "%s", message)
	}

	log.Error(message, append(fields, zap.Error(err))...)
	return status.Errorf(code, "%s: %s", message, err)
}

func buildNoteIdFromCollectorId(collectorId string, note *grafeas_proto.Note) string {
	switch note.Kind {
	case common_go_proto.NoteKind_VULNERABILITY:
		return fmt.Sprintf("%s-vulnerability", collectorId)
	case common_go_proto.NoteKind_BUILD:
		return fmt.Sprintf("%s-build", collectorId)
	case common_go_proto.NoteKind_IMAGE:
		return fmt.Sprintf("%s-image", collectorId)
	case common_go_proto.NoteKind_PACKAGE:
		return fmt.Sprintf("%s-package", collectorId)
	case common_go_proto.NoteKind_DEPLOYMENT:
		return fmt.Sprintf("%s-deployment", collectorId)
	case common_go_proto.NoteKind_DISCOVERY:
		return fmt.Sprintf("%s-discovery", collectorId)
	case common_go_proto.NoteKind_ATTESTATION:
		return fmt.Sprintf("%s-attestation", collectorId)
	case common_go_proto.NoteKind_INTOTO:
		return fmt.Sprintf("%s-intoto", collectorId)
	}

	return fmt.Sprintf("%s-unspecified", collectorId)
}

func getNoteIdFromNoteName(noteName string) string {
	// note name format: projects/${projectId}/notes/${noteId}
	return strings.TrimPrefix(noteName, constants.RodeProjectSlug+"/notes/")
}
