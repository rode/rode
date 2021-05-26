package policy

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

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
		grafeasClient *mocks.FakeGrafeasV1Beta1Client
		indexManager  *immocks.FakeIndexManager

		manager Manager
	)

	BeforeEach(func() {
		esClient = &esutilfakes.FakeClient{}
		grafeasClient = &mocks.FakeGrafeasV1Beta1Client{}
		indexManager = &immocks.FakeIndexManager{}
		c := &config.ElasticsearchConfig{
			Refresh: "true",
		}

		expectedPoliciesAlias = fake.LetterN(10)
		indexManager.AliasNameReturns(expectedPoliciesAlias)

		manager = NewManager(logger, esClient, c, indexManager, nil, nil, grafeasClient)
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
