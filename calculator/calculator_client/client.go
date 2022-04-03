package main

import (
	"context"
	"fmt"
	"go-grpc-course-interactive/calculator/calculatorpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"time"
)

func main() {
	conn, err := grpc.Dial(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Panicln("could not connect", err)
	}
	defer conn.Close()

	client := calculatorpb.NewCalculatorServiceClient(conn)
	fmt.Println("created client", client)

	// doSum(client)

	// doComputeAverage(client)

	// doFindMaximum(client)

	doErrorUnary(client)
}

func doSum(client calculatorpb.CalculatorServiceClient) {
	req := &calculatorpb.SumRequest{
		Arg1: 3,
		Arg2: 10,
	}
	res, err := client.Sum(context.Background(), req)
	if err != nil {
		log.Panicln("error while calling Sum rpc", err)
	}
	log.Println("response from Sum:", res.Sum)
}

func doComputeAverage(client calculatorpb.CalculatorServiceClient) {
	log.Println("starting compute average streaming rpc...")
	requests := []*calculatorpb.ComputeAverageRequest{
		{
			Arg: 1,
		},
		{
			Arg: 2,
		},
		{
			Arg: 3,
		},
		{
			Arg: 4,
		},
	}
	stream, err := client.ComputeAverage(context.Background())
	if err != nil {
		log.Panicln("error while calling ComputeAverage", err)
	}

	for _, req := range requests {
		err := stream.Send(req)
		if err != nil {
			log.Panicln("error sending client stream", err)
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Panicln("error receiving server response from ComputeAverage", err)
	}
	log.Println("ComputeAverage response", res.Average)
}

func doFindMaximum(client calculatorpb.CalculatorServiceClient) {
	fmt.Println("starting find maximum rpc...")
	numbers := []int32{1, 5, 3, 6, 2, 20}
	stream, err := client.FindMaximum(context.Background())
	if err != nil {
		log.Panicln("error while creating stream", err)
		return
	}
	done := make(chan struct{})
	go func() {
		for _, number := range numbers {
			log.Println("sending", number)
			err := stream.Send(&calculatorpb.FindMaximumRequest{
				Arg: number,
			})
			if err == io.EOF {
				close(done)
			}
			if err != nil {
				log.Panicln("error sending request", err)
				return
			}
			time.Sleep(250 * time.Millisecond)
		}
		err = stream.CloseSend()
		if err != nil {
			log.Panicln("error closing client stream", err)
			return
		}
	}()

	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Panicln("error receiving request", err)
			}
			fmt.Println("received", res.Max)
		}
		close(done)
	}()

	<-done
}

func doErrorUnary(client calculatorpb.CalculatorServiceClient) {
	fmt.Println("starting square root rpc...")
	number := int32(-4)
	res, err := client.SquareRoot(context.Background(), &calculatorpb.SquareRootRequest{
		Number: number,
	})
	if err != nil {
		respError, ok := status.FromError(err)
		if ok {
			// grpc error
			log.Println(respError.Code(), respError.Message())
			if respError.Code() == codes.InvalidArgument {
				log.Println("Sent a bad argument", respError.Message())
				return
			}
		} else {
			log.Panicln("error calling SquareRoot", err)
		}

		return
	}
	log.Println("Sqrt of", number, "is", res.GetSqrt())

}
