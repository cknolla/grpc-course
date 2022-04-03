package main

import (
	"context"
	"go-grpc-course-interactive/blog/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"path/filepath"
)

func main() {
	var err error
	log.Println("starting blog client")
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

	client := pb.NewBlogServiceClient(conn)
	log.Println("created client", client)

	blog := &pb.Blog{
		AuthorId: "Bob",
		Title:    "A Title",
		Content:  "Some content",
	}
	response, err := client.CreateBlog(context.Background(), &pb.CreateBlogRequest{Blog: blog})
	if err != nil {
		log.Println("error creating blog", err)
	}
	log.Println("blog created", response)

	// read blog
	log.Println("reading blog with bad id")
	_, err = client.ReadBlog(context.Background(), &pb.ReadBlogRequest{Id: "123jlk13j12"})
	if err != nil {
		log.Println("error reading blog", err)
	}
	log.Println("reading blog")
	readResponse, err := client.ReadBlog(context.Background(), &pb.ReadBlogRequest{Id: response.GetBlog().GetId()})
	if err != nil {
		log.Println("error reading blog", err)
	}
	log.Println("received blog", readResponse.GetBlog())

	// update blog
	updatedBlog := &pb.Blog{
		Id:       response.GetBlog().GetId(),
		AuthorId: "Changed Author",
		Title:    "Changed Title",
		Content:  "Changed content",
	}
	log.Println("updating blog")
	updateRes, err := client.UpdateBlog(context.Background(), &pb.UpdateBlogRequest{
		Blog: updatedBlog,
	})
	if err != nil {
		log.Println("error updating blog", err)
	}
	log.Println("blog was updated", updateRes.GetBlog())

}
