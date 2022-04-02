package main

import (
	"context"
	"fmt"
	"go-grpc-course-interactive/calculator/calculatorpb"
	"google.golang.org/grpc"
	"log"
	"net"
)

type server struct{}

func (*server) Sum(ctx context.Context, req *calculatorpb.SumRequest) (*calculatorpb.SumResponse, error) {
	fmt.Println("Sum function was invoked with", req)
	// fetch data from the passed-in request
	arg1 := req.Arg1
	arg2 := req.Arg2
	sum := arg1 + arg2
	res := &calculatorpb.SumResponse{
		Sum: sum,
	}
	return res, nil
}

func main() {
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Println("Failed to listen:", err)
	}

	s := grpc.NewServer()
	calculatorpb.RegisterSumServiceServer(s, &server{})

	if err := s.Serve(lis); err != nil {
		log.Panicln("Failed to serve:", err)
	}
}
