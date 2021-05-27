package policy

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	immocks "github.com/rode/es-index-manager/mocks"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil/esutilfakes"
	"github.com/rode/rode/config"
	"github.com/rode/rode/mocks"
	pb "github.com/rode/rode/proto/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	//go:embed test/good.rego
	goodPolicy string
	//go:embed test/missing_rode_fields.rego
	compilablePolicyMissingRodeFields string
	//go:embed test/missing_results_fields.rego
	compilablePolicyMissingResultsFields string
	//go:embed test/missing_results_return.rego
	compilablePolicyMissingResultsReturn string
	//go:embed test/uncompilable.rego
	uncompilablePolicy string
	unparseablePolicy  = `
		package play
		default hello = false
		hello
			m := input.message
			m == "world"
		}`
	invalidJson = []byte{'}'}
)

var _ = Describe("PolicyManager", func() {
	var (
		ctx                   = context.Background()
		expectedPoliciesAlias string

		esClient      *esutilfakes.FakeClient
		esConfig      *config.ElasticsearchConfig
		grafeasClient *mocks.FakeGrafeasV1Beta1Client
		indexManager  *immocks.FakeIndexManager

		manager Manager
	)

	BeforeEach(func() {
		esClient = &esutilfakes.FakeClient{}
		grafeasClient = &mocks.FakeGrafeasV1Beta1Client{}
		indexManager = &immocks.FakeIndexManager{}
		esConfig = &config.ElasticsearchConfig{
			Refresh: config.RefreshOption(fake.RandomString([]string{config.RefreshTrue, config.RefreshFalse, config.RefreshWaitFor})),
		}

		expectedPoliciesAlias = fake.LetterN(10)
		indexManager.AliasNameReturns(expectedPoliciesAlias)

		manager = NewManager(logger, esClient, esConfig, indexManager, nil, nil, grafeasClient)
	})

	Context("CreatePolicy", func() {
		var (
			policyId        string
			policyVersionId string
			version         int32
			policy          *pb.Policy
			policyEntity    *pb.PolicyEntity

			bulkResponse      *esutil.EsBulkResponse
			bulkResponseError error

			actualPolicy *pb.Policy
			actualError  error
		)

		BeforeEach(func() {
			policyId = fake.UUID()
			version = 1 // initial version is always 1
			newUuid = func() uuid.UUID {
				return uuid.MustParse(policyId)
			}
			policyVersionId = fmt.Sprintf("%s.%d", policyId, version)

			policy = createRandomPolicy(fake.UUID(), version)
			policyEntity = createRandomPolicyEntity(goodPolicy, version)

			policy.Policy = policyEntity

			bulkResponse = &esutil.EsBulkResponse{
				Errors: false,
				Items: []*esutil.EsBulkResponseItem{
					{
						Create: &esutil.EsIndexDocResponse{
							Status: http.StatusOK,
						},
					},
					{
						Create: &esutil.EsIndexDocResponse{
							Status: http.StatusOK,
						},
					},
				},
			}
			bulkResponseError = nil
		})

		JustBeforeEach(func() {
			if esClient.BulkStub == nil {
				esClient.BulkReturns(bulkResponse, bulkResponseError)
			}

			actualPolicy, actualError = manager.CreatePolicy(ctx, deepCopyPolicy(policy))
		})

		When("the policy is valid", func() {
			It("should not return an error", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("should send a bulk request to create the policy and its initial version", func() {
				Expect(esClient.BulkCallCount()).To(Equal(1))

				_, actualRequest := esClient.BulkArgsForCall(0)

				Expect(actualRequest.Index).To(Equal(expectedPoliciesAlias))
				Expect(actualRequest.Refresh).To(Equal(esConfig.Refresh.String()))
				Expect(actualRequest.Items).To(HaveLen(2))

				createPolicyItem := actualRequest.Items[0]
				Expect(createPolicyItem.Operation).To(Equal(esutil.BULK_CREATE))
				Expect(createPolicyItem.DocumentId).To(Equal(policyId))
				Expect(createPolicyItem.Join.Field).To(Equal("join"))
				Expect(createPolicyItem.Join.Name).To(Equal("policy"))

				createPolicyEntityItem := actualRequest.Items[1]
				Expect(createPolicyEntityItem.Operation).To(Equal(esutil.BULK_CREATE))
				Expect(createPolicyEntityItem.DocumentId).To(Equal(policyVersionId))
				Expect(createPolicyEntityItem.Join.Field).To(Equal("join"))
				Expect(createPolicyEntityItem.Join.Name).To(Equal("version"))
			})

			Describe("policy content in parent document", func() {
				var actualPolicyMessage *pb.Policy

				BeforeEach(func() {
					esClient.BulkCalls(func(ctx context.Context, request *esutil.BulkRequest) (*esutil.EsBulkResponse, error) {
						message, ok := request.Items[0].Message.(*pb.Policy)
						Expect(ok).To(BeTrue())
						actualPolicyMessage = deepCopyPolicy(message)

						return bulkResponse, bulkResponseError
					})
				})

				It("should remove the policy content from the parent document", func() {
					Expect(actualPolicyMessage).NotTo(BeNil())
					Expect(actualPolicyMessage.Policy).To(BeNil())
				})
			})

			It("should return the policy at its current version", func() {
				Expect(actualPolicy.Id).To(Equal(policyId))
				Expect(actualPolicy.Name).To(Equal(policy.Name))
				Expect(actualPolicy.Description).To(Equal(policy.Description))
				Expect(actualPolicy.CurrentVersion).To(Equal(version))
				Expect(actualPolicy.Created.IsValid()).To(BeTrue())
				Expect(actualPolicy.Updated.IsValid()).To(BeTrue())

				Expect(actualPolicy.Policy).NotTo(BeNil())
				Expect(actualPolicy.Policy.RegoContent).To(Equal(policyEntity.RegoContent))
				Expect(actualPolicy.Policy.SourcePath).To(Equal(policyEntity.SourcePath))
				Expect(actualPolicy.Policy.Created).To(Equal(actualPolicy.Created))
				Expect(actualPolicy.Policy.Version).To(Equal(version))
				Expect(actualPolicy.Policy.Message).To(Equal("Initial policy creation"))
			})
		})

		When("the policy is invalid", func() {
			BeforeEach(func() {
				policy.Policy.RegoContent = unparseablePolicy
			})

			It("should return an error with details", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())

				status := getGRPCStatusFromError(actualError)
				Expect(status.Code()).To(Equal(codes.InvalidArgument))

				Expect(status.Details()).To(HaveLen(1))
				detailsMsg := status.Details()[0].(*pb.ValidatePolicyResponse)

				Expect(detailsMsg.Policy).To(Equal(policyEntity.RegoContent))
				Expect(detailsMsg.Compile).To(BeFalse())
				Expect(detailsMsg.Errors).To(HaveLen(1))
			})

			It("should not create any documents", func() {
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("validating the policy causes an error", func() {
			BeforeEach(func() {
				policy.Policy.RegoContent = unparseablePolicy
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())

				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not create any documents", func() {
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("the policy name is unset", func() {
			BeforeEach(func() {
				policy.Name = ""
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())

				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not create any documents", func() {
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("the bulk create fails", func() {
			BeforeEach(func() {
				bulkResponseError = errors.New("bulk error")
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())

				status := getGRPCStatusFromError(actualError)
				Expect(status.Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy creation in the bulk request fails", func() {
			BeforeEach(func() {
				bulkResponse.Items[0].Create.Status = http.StatusInternalServerError
				bulkResponse.Items[0].Create.Error = &esutil.EsIndexDocError{
					Type:   fake.Word(),
					Reason: fake.Word(),
				}
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())

				status := getGRPCStatusFromError(actualError)
				Expect(status.Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy entity creation in the bulk request fails", func() {
			BeforeEach(func() {
				bulkResponse.Items[1].Create.Status = http.StatusInternalServerError
				bulkResponse.Items[1].Create.Error = &esutil.EsIndexDocError{
					Type:   fake.Word(),
					Reason: fake.Word(),
				}
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())

				status := getGRPCStatusFromError(actualError)
				Expect(status.Code()).To(Equal(codes.Internal))
			})
		})
	})

	Context("GetPolicy", func() {
		var (
			policyId             string
			policyVersionId      string
			version              int32
			request              *pb.GetPolicyRequest
			expectedPolicy       *pb.Policy
			expectedPolicyEntity *pb.PolicyEntity

			actualError  error
			actualPolicy *pb.Policy

			getPolicyResponse *esutil.EsGetResponse
			getPolicyError    error

			getPolicyEntityResponse *esutil.EsGetResponse
			getPolicyEntityError    error
		)

		BeforeEach(func() {
			policyId = fake.UUID()
			version = int32(fake.Number(1, 10))
			policyVersionId = fmt.Sprintf("%s.%d", policyId, version)

			request = &pb.GetPolicyRequest{
				Id: policyId,
			}

			expectedPolicy = createRandomPolicy(policyId, version)
			policyJson, _ := protojson.Marshal(expectedPolicy)

			getPolicyResponse = &esutil.EsGetResponse{
				Id:     policyId,
				Found:  true,
				Source: policyJson,
			}
			getPolicyError = nil

			expectedPolicyEntity = createRandomPolicyEntity(goodPolicy, version)
			policyEntityJson, _ := protojson.Marshal(expectedPolicyEntity)
			getPolicyEntityResponse = &esutil.EsGetResponse{
				Id:     policyVersionId,
				Found:  true,
				Source: policyEntityJson,
			}
			getPolicyEntityError = nil
		})

		JustBeforeEach(func() {
			esClient.GetReturnsOnCall(0, getPolicyResponse, getPolicyError)
			esClient.GetReturnsOnCall(1, getPolicyEntityResponse, getPolicyEntityError)

			actualPolicy, actualError = manager.GetPolicy(ctx, request)
		})

		When("the policy exists", func() {
			It("should not return an error", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("should query Elasticsearch for the policy", func() {
				Expect(indexManager.AliasNameCallCount()).To(Equal(2))

				actualDocumentKind, inner := indexManager.AliasNameArgsForCall(0)
				Expect(actualDocumentKind).To(Equal("policies"))
				Expect(inner).To(Equal(""))

				Expect(esClient.GetCallCount()).To(Equal(2))

				_, actualRequest := esClient.GetArgsForCall(0)

				Expect(actualRequest.Index).To(Equal(expectedPoliciesAlias))
				Expect(actualRequest.DocumentId).To(Equal(policyId))
			})

			It("should query Elasticsearch for the versioned policy entity", func() {
				_, actualRequest := esClient.GetArgsForCall(1)

				Expect(actualRequest.Index).To(Equal(expectedPoliciesAlias))
				Expect(actualRequest.DocumentId).To(Equal(policyVersionId))
				Expect(actualRequest.Join.Parent).To(Equal(policyId))
				Expect(actualRequest.Join.Field).To(Equal("join"))
				Expect(actualRequest.Join.Name).To(Equal("version"))
			})

			It("should return the policy at its current version", func() {
				Expect(actualPolicy).NotTo(BeNil())
				Expect(actualPolicy.Id).To(Equal(policyId))
				Expect(actualPolicy.Name).To(Equal(expectedPolicy.Name))
				Expect(actualPolicy.Description).To(Equal(expectedPolicy.Description))
				Expect(actualPolicy.CurrentVersion).To(Equal(version))

				Expect(actualPolicy.Policy).NotTo(BeNil())
				Expect(actualPolicy.Policy.Version).To(Equal(version))
				Expect(actualPolicy.Policy.RegoContent).To(Equal(goodPolicy))
			})
		})

		When("an error occurs fetching policy", func() {
			BeforeEach(func() {
				getPolicyError = errors.New("get policy error")
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not try to fetch the policy entity", func() {
				Expect(esClient.GetCallCount()).To(Equal(1))
			})
		})

		When("the policy is not found", func() {
			BeforeEach(func() {
				getPolicyResponse.Found = false
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.NotFound))
			})

			It("should not try to fetch the policy entity", func() {
				Expect(esClient.GetCallCount()).To(Equal(1))
			})
		})

		When("the policy document is invalid", func() {
			BeforeEach(func() {
				getPolicyResponse.Source = invalidJson
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not try to fetch the policy version", func() {
				Expect(esClient.GetCallCount()).To(Equal(1))
			})
		})

		When("an error occurs fetching the policy entity", func() {
			BeforeEach(func() {
				getPolicyEntityError = errors.New("get policy entity error")
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy entity is not found", func() {
			BeforeEach(func() {
				getPolicyEntityResponse.Found = false
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy entity document is invalid", func() {
			BeforeEach(func() {
				getPolicyEntityResponse.Source = invalidJson
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})
	})

	Context("ValidatePolicy", func() {
		var (
			request *pb.ValidatePolicyRequest

			actualResponse *pb.ValidatePolicyResponse
			actualError    error
		)

		BeforeEach(func() {
			request = &pb.ValidatePolicyRequest{
				Policy: goodPolicy,
			}
		})

		JustBeforeEach(func() {
			actualResponse, actualError = manager.ValidatePolicy(ctx, request)
		})

		When("the policy is valid", func() {
			It("should not return an error", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("should indicate successful compilation in the response", func() {
				Expect(actualResponse.Compile).To(BeTrue())
			})

			It("should not return any policy errors", func() {
				Expect(actualResponse.Errors).To(BeEmpty())
			})
		})

		When("the policy is empty", func() {
			BeforeEach(func() {
				request.Policy = ""
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})
		})

		When("the policy fails to compile", func() {
			BeforeEach(func() {
				request.Policy = uncompilablePolicy
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should indicate that compilation failed in the response", func() {
				Expect(actualResponse.Compile).To(BeFalse())
			})

			It("should return the compilation errors", func() {
				Expect(len(actualResponse.Errors)).To(BeNumerically(">", 0))
			})
		})

		When("the policy is missing required fields in the result", func() {
			BeforeEach(func() {
				request.Policy = compilablePolicyMissingResultsFields
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should include an error message about the missing field", func() {
				Expect(actualResponse.Errors).To(HaveLen(1))
			})
		})

		When("the policy does not contain a rule that returns results", func() {
			BeforeEach(func() {
				request.Policy = compilablePolicyMissingResultsReturn
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should include an error message about the missing result", func() {
				Expect(actualResponse.Errors).To(HaveLen(1))
			})
		})

		When("the policy does not have pass or violations rules", func() {
			BeforeEach(func() {
				request.Policy = compilablePolicyMissingRodeFields
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should include an error message about the missing rules", func() {
				Expect(actualResponse.Errors).To(HaveLen(3))
			})
		})

		When("the policy cannot be parsed", func() {
			BeforeEach(func() {
				request.Policy = unparseablePolicy
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should include an error message", func() {
				Expect(actualResponse.Errors).To(HaveLen(1))
			})

		})
	})
})

func createRandomPolicy(id string, version int32) *pb.Policy {
	return &pb.Policy{
		Id:             id,
		Name:           fake.Word(),
		Description:    fake.Word(),
		CurrentVersion: version,
	}
}

func createRandomPolicyEntity(policy string, version int32) *pb.PolicyEntity {
	return &pb.PolicyEntity{
		Version:     version,
		RegoContent: policy,
		SourcePath:  fake.URL(),
		Message:     fake.Word(),
	}
}

func getGRPCStatusFromError(err error) *status.Status {
	s, ok := status.FromError(err)
	Expect(ok).To(BeTrue(), "Expected error to be a gRPC status")

	return s
}

func deepCopyPolicy(policy *pb.Policy) *pb.Policy {
	policyJson, err := protojson.Marshal(policy)
	Expect(err).NotTo(HaveOccurred())

	var newPolicy pb.Policy
	Expect(protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(policyJson, &newPolicy)).NotTo(HaveOccurred())

	return &newPolicy
}
