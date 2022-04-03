package main

import (
	"context"
	"fmt"
	"go-grpc-course-interactive/greet/greetpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"path/filepath"
	"time"
)

func main() {
	var err error
	tls := false
	creds := insecure.NewCredentials()
	if tls {
		creds, err = credentials.NewClientTLSFromFile(
			filepath.Join("ssl", "ca.crt"),
			"",
		)
		if err != nil {
			log.Panicln("error loading credentials", err)
		}
	}

	conn, err := grpc.Dial(
		"localhost:50051",
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		log.Panicln("could not connect", err)
	}
	defer conn.Close()

	client := greetpb.NewGreetServiceClient(conn)
	fmt.Printf("created client: %f", client)

	doUnary(client)

	// doServerStreaming(client)

	// doClientStreaming(client)

	// doBiDiStreaming(client)
	// doUnaryWithDeadline(client, 5*time.Second) // should complete
	// doUnaryWithDeadline(client, 1*time.Second) // should timeout
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

func doClientStreaming(client greetpb.GreetServiceClient) {
	fmt.Println("starting to do a client streaming rpc...")
	requests := []*greetpb.LongGreetRequest{
		{
			Greeting: &greetpb.Greeting{
				FirstName: "Bob",
				LastName:  "Loblaw",
			},
		},
		{
			Greeting: &greetpb.Greeting{
				FirstName: "James",
				LastName:  "Bond",
			},
		},
		{
			Greeting: &greetpb.Greeting{
				FirstName: "Margaret",
				LastName:  "Thatcher",
			},
		},
	}
	stream, err := client.LongGreet(context.Background())
	if err != nil {
		log.Panicln("error while calling LongGreet", err)
	}

	// iterate over req slice and send each message
	for _, req := range requests {
		err := stream.Send(req)
		if err != nil {
			log.Panicln("error sending client stream", err)
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Panicln("error receiving server response from LongGreet", err)
	}
	log.Println("LongGreet response", res.GetResult())
}

func doBiDiStreaming(client greetpb.GreetServiceClient) {
	fmt.Println("starting bidi streaming rpc...")
	requests := []*greetpb.GreetEveryoneRequest{
		{
			Greeting: &greetpb.Greeting{
				FirstName: "Bob",
				LastName:  "Loblaw",
			},
		},
		{
			Greeting: &greetpb.Greeting{
				FirstName: "James",
				LastName:  "Bond",
			},
		},
		{
			Greeting: &greetpb.Greeting{
				FirstName: "Margaret",
				LastName:  "Thatcher",
			},
		},
	}
	// create a stream by invoking client
	stream, err := client.GreetEveryone(context.Background())
	if err != nil {
		log.Panicln("error while creating stream", err)
		return
	}
	done := make(chan struct{})
	// send a bunch of messages
	go func() {
		for _, req := range requests {
			log.Println("sending message", req)
			err := stream.Send(req)
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

	// receive a bunch of messages
	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Panicln("error while receiving request", err)
				return
			}
			fmt.Println("received", res.Result)
		}
		close(done)
	}()

	<-done
}

func doUnaryWithDeadline(client greetpb.GreetServiceClient, waitTime time.Duration) {
	fmt.Println("starting to do unary with deadline rpc...")
	req := &greetpb.GreetWithDeadlineRequest{
		Greeting: &greetpb.Greeting{
			FirstName: "Bob",
			LastName:  "What",
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), waitTime)
	defer cancel()

	res, err := client.GreetWithDeadline(ctx, req)
	if err != nil {
		statusError, ok := status.FromError(err)
		if ok {
			if statusError.Code() == codes.DeadlineExceeded {
				fmt.Println("Timeout was hit. Deadline exceeded.")
			} else {
				fmt.Println("unexpected error", statusError)
			}
		} else {
			log.Panicln("error while calling Greet rpc", err)
		}
		return
	}
	log.Println("Response from Greet:", res.Result)
}
