package main

import (
	"context"
	"fmt"
	"go-grpc-course-interactive/prime/primepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
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

	client := primepb.NewPrimeServiceClient(conn)
	fmt.Println("created client", client)

	fmt.Println("beginning prime decomposition rpc...")
	req := &primepb.PrimeRequest{
		Arg: 5684921,
	}

	resStream, err := client.Prime(context.Background(), req)
	if err != nil {
		log.Panicln("err while calling Prime", err)
	}
	for {
		msg, err := resStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Panicln("error while reading stream", err)
		}
		log.Println(msg.GetPrime())
	}
}
