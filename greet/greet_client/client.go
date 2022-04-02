package main

import (
	"context"
	"fmt"
	"go-grpc-course-interactive/greet/greetpb"
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

	client := greetpb.NewGreetServiceClient(conn)
	fmt.Printf("created client: %f", client)

	// doUnary(client)

	doServerStreaming(client)
}

func doUnary(client greetpb.GreetServiceClient) {
	fmt.Println("starting to do unary rpc...")
	req := &greetpb.GreetRequest{
		Greeting: &greetpb.Greeting{
			FirstName: "Bob",
			LastName:  "What",
		},
	}
	res, err := client.Greet(context.Background(), req)
	if err != nil {
		log.Panicln("error while calling Greet rpc", err)
	}
	log.Println("Response from Greet:", res.Result)
}

func doServerStreaming(client greetpb.GreetServiceClient) {
	fmt.Println("starting to do a server streaming rpc...")
	req := &greetpb.GreetManyTimesRequest{
		Greeting: &greetpb.Greeting{
			FirstName: "Bob",
			LastName:  "What",
		},
	}
	resStream, err := client.GreetManyTimes(context.Background(), req)
	if err != nil {
		log.Panicln("err while calling GreetManyTimes rpc", err)
	}
	for {
		msg, err := resStream.Recv()
		if err == io.EOF {
			// server closed stream
			break
		}
		if err != nil {
			log.Panicln("error while reading stream", err)
		}
		log.Println("Response from GreetManyTimes", msg.GetResult())
	}
}
