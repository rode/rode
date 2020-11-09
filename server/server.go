package server

import (
	"context"
	"fmt"
	pb "github.com/liatrio/rode-api/proto/v1alpha1"
	grafeas "github.com/liatrio/rode-api/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
)

type rodeServer struct {
	pb.UnimplementedRodeServer
	grafeasClient grafeas.GrafeasV1Beta1Client
}

func (r *rodeServer) BatchCreateOccurrences(ctx context.Context, occurrenceRequest *grafeas.BatchCreateOccurrencesRequest) (*grafeas.BatchCreateOccurrencesResponse, error) {
	fmt.Println("This works!!!!!")
	fmt.Printf("%#v\n", *occurrenceRequest.Occurrences[0])

	//Forward to grafeas to create occurrence
	occurrenceResponse, err := r.grafeasClient.BatchCreateOccurrences(ctx, occurrenceRequest)
	if err != nil {
		fmt.Println("Failed to create occurrence")
		return nil, err
	}
	return occurrenceResponse, nil
}

func NewRodeServer(grafeasClient grafeas.GrafeasV1Beta1Client) pb.RodeServer {
	return &rodeServer{
		grafeasClient: grafeasClient,
	}
}
