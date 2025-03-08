package server

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/denysvitali/odi-backend/gen/proto"
)

type Server struct {
	proto.UnimplementedOdiServiceServer
}

var (
	log = logrus.StandardLogger()
)

func New() *Server {
	return &Server{}
}

func (s *Server) Listen(grpcListenAddr string, httpListenAddr string) error {
	// Start GRPC server
	grpcServer := grpc.NewServer()
	grpcServer.RegisterService(&proto.OdiService_ServiceDesc, s)
	listener, err := net.Listen("tcp", grpcListenAddr)
	if err != nil {
		return fmt.Errorf("listen: %v", err)
	}
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = proto.RegisterOdiServiceHandlerFromEndpoint(ctx, mux, grpcListenAddr, opts)
	if err != nil {
		return err
	}
	return http.ListenAndServe(httpListenAddr, mux)
}

func (s *Server) GetDocument(ctx context.Context, req *proto.GetDocumentRequest) (*proto.GetDocumentResponse, error) {
	// TODO: Implement
	log.Infof("GetDocument: %v", req)
	return nil, nil
}
