package main

import (
	"context"
	"fmt"
	"go-grpc-course-interactive/blog/pb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
)

var collection *mongo.Collection

type server struct{}

type blogItem struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	AuthorId string             `bson:"author_id"`
	Content  string             `bson:"content"`
	Title    string             `bson:"title"`
}

func (*server) CreateBlog(ctx context.Context, req *pb.CreateBlogRequest) (*pb.CreateBlogResponse, error) {
	blog := req.GetBlog()

	data := blogItem{
		AuthorId: blog.GetAuthorId(),
		Content:  blog.GetContent(),
		Title:    blog.GetTitle(),
	}

	log.Println("inserting blog", data)
	res, err := collection.InsertOne(context.Background(), data)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.CreateBlogResponse{
		Blog: &pb.Blog{
			Id:       id.Hex(),
			AuthorId: blog.GetAuthorId(),
			Title:    blog.GetTitle(),
			Content:  blog.GetContent(),
		},
	}, nil
}

// findById fetches a single blog item by object id.
func findById(objectId primitive.ObjectID) (*blogItem, error) {
	data := &blogItem{}
	filter := bson.M{"_id": objectId}
	log.Println("getting blog with id", objectId)
	res := collection.FindOne(context.Background(), filter)
	if err := res.Decode(data); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			"cannot find blog with id",
			objectId,
		)
	}
	return data, nil
}

func (*server) ReadBlog(ctx context.Context, req *pb.ReadBlogRequest) (*pb.ReadBlogResponse, error) {
	objectId, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	data, err := findById(objectId)
	if err != nil {
		return nil, err
	}
	return &pb.ReadBlogResponse{
		Blog: &pb.Blog{
			Id:       data.ID.Hex(),
			AuthorId: data.AuthorId,
			Content:  data.Content,
			Title:    data.Title,
		},
	}, nil
}

func (*server) UpdateBlog(ctx context.Context, req *pb.UpdateBlogRequest) (*pb.UpdateBlogResponse, error) {
	blog := req.GetBlog()
	objectId, err := primitive.ObjectIDFromHex(blog.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	data, err := findById(objectId)
	if err != nil {
		return nil, err
	}
	data.AuthorId = blog.GetAuthorId()
	data.Content = blog.GetContent()
	data.Title = blog.GetTitle()
	log.Println("updating blog with id", objectId)
	_, err = collection.ReplaceOne(context.Background(), bson.M{"_id": objectId}, data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"cannot update object in mongo: %v", err)
	}
	return &pb.UpdateBlogResponse{
		Blog: &pb.Blog{
			Id:       data.ID.Hex(),
			AuthorId: data.AuthorId,
			Content:  data.Content,
			Title:    data.Title,
		},
	}, nil
}

func main() {
	// more detail on crash
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	mongoConfig := map[string]string{
		"host": "localhost",
		"port": "27017",
	}
	log.Println("connecting to mongo")
	mongoClientOptions := options.Client().ApplyURI(
		fmt.Sprintf("mongodb://%s:%s",
			mongoConfig["host"],
			mongoConfig["port"],
		),
	)
	mongoClient, err := mongo.Connect(context.TODO(), mongoClientOptions)
	if err != nil {
		log.Panicln("error connecting to mongo:", err)
	}
	// defer func() {
	// 	log.Println("disconnecting mongo")
	// 	_ = mongoClient.Disconnect(context.TODO())
	// }()
	err = mongoClient.Ping(context.TODO(), nil)
	if err != nil {
		log.Panicln("error pinging mongo:", err)
	}
	log.Println("mongo connection succeeded")

	collection = mongoClient.Database("blog").Collection("blogs")

	log.Println("init BlogService")

	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Panicln("failed to listen:", err)
	}
	// defer func() {
	// 	log.Println("closing listener")
	// 	_ = lis.Close()
	// }()
	tls := false
	creds := insecure.NewCredentials()
	if tls {
		creds, err = credentials.NewServerTLSFromFile(
			filepath.Join("ssl", "server.crt"),
			filepath.Join("ssl", "server.pem"),
		)
		if err != nil {
			log.Panicln("error loading credentials:", err)
		}
	}

	s := grpc.NewServer(grpc.Creds(creds))
	// defer func() {
	// 	log.Println("stopping server")
	// 	s.Stop()
	// }()
	pb.RegisterBlogServiceServer(s, &server{})

	go func() {
		log.Println("starting server and listening for requests")
		if err := s.Serve(lis); err != nil {
			log.Panicln("failed to serve:", err)
		}
	}()

	// wait for ctrl+c
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	// block until signal received
	<-ch
	log.Println("stopping server")
	s.Stop()
	log.Println("closing listener")
	_ = lis.Close()
	log.Println("disconnecting mongo")
	_ = mongoClient.Disconnect(context.TODO())
	log.Println("exiting...")
}
