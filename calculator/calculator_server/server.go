package main

import (
	"context"
	"fmt"
	"go-grpc-course-interactive/calculator/calculatorpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"math"
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

func (*server) ComputeAverage(stream calculatorpb.CalculatorService_ComputeAverageServer) error {
	log.Println("ComputeAverage invoked with a stream request")
	sum := 0.0
	for i := 0; ; i++ {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&calculatorpb.ComputeAverageResponse{
				Average: sum / float64(i),
			})
		}
		if err != nil {
			log.Panicln("error while reading client stream", err)
		}
		sum += float64(req.Arg)
	}
}

func (*server) FindMaximum(stream calculatorpb.CalculatorService_FindMaximumServer) error {
	log.Println("FindMaximum invoked with a streaming request")
	max := -(int32(^uint32(0)>>1) - 1) // smallest possible int
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Panicln("error while reading client stream", err)
			return err
		}
		number := req.GetArg()
		if number > max {
			max = number
			err = stream.Send(&calculatorpb.FindMaximumResponse{
				Max: max,
			})
			if err != nil {
				log.Panicln("error while sending data to client", err)
				return err
			}
		}
	}
}

func (*server) SquareRoot(ctx context.Context, req *calculatorpb.SquareRootRequest) (*calculatorpb.SquareRootResponse, error) {
	fmt.Println("received SquareRoot rpc")
	number := req.GetNumber()
	if number < 0 {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Received a negative number: %v",
			number,
		)
	}
	return &calculatorpb.SquareRootResponse{
		Sqrt: math.Sqrt(float64(number)),
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Println("Failed to listen:", err)
	}

	s := grpc.NewServer()
	calculatorpb.RegisterCalculatorServiceServer(s, &server{})

	// register reflection for client discovery
	reflection.Register(s)

	if err := s.Serve(lis); err != nil {
		log.Panicln("Failed to serve:", err)
	}
}
