package main

import (
	"fmt"
	"go-grpc-course-interactive/prime/primepb"
	"google.golang.org/grpc"
	"log"
	"net"
)

type server struct{}

func (*server) Prime(
	req *primepb.PrimeRequest,
	stream primepb.PrimeService_PrimeServer,
) error {
	log.Println("Prime invoked with ", req)
	remainder := req.Arg
	for divisor := int32(2); remainder > 1; divisor++ {
		for remainder%divisor == 0 {
			remainder /= divisor
			res := &primepb.PrimeResponse{
				Prime: divisor,
			}
			err := stream.Send(res)
			if err != nil {
				log.Panicln("failed to send stream", err)
			}
		}
	}
	return nil
}

func main() {
	fmt.Println("starting prime stream server")

	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	primepb.RegisterPrimeServiceServer(s, &server{})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
