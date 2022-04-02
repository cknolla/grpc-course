package main

import (
	"context"
	"fmt"
	"go-grpc-course-interactive/calculator/calculatorpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
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

	client := calculatorpb.NewSumServiceClient(conn)
	fmt.Println("created client", client)

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
