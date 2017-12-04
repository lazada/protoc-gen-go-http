package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/lazada/protoc-gen-go-http/codec"
	pb "github.com/lazada/protoc-gen-go-http/example/pb"
	"google.golang.org/grpc"
)

type server struct{}

func (s *server) GetPerson(ctx context.Context, query *pb.Query) (*pb.Person, error) {
	return &pb.Person{}, nil
}

func (s *server) ListPeople(query *pb.Query, server pb.Example_ListPeopleServer) error {
	return nil
}

func main() {
	s := grpc.NewServer()
	srv := &server{}
	pb.RegisterExampleServer(s, srv)

	router, err := pb.NewExampleRouter(srv, func() codec.Codec {
		return codec.NewRESTCCodec()
	}, pb.WithSwagger())
	if err != nil {
		panic(err)
	}

	go func() {
		if err := http.ListenAndServe(":8080", router); err != nil {
			panic(err)
		}
	}()

	lis, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
