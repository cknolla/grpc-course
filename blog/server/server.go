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
	"google.golang.org/grpc/reflection"
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
func findById(id string) (*blogItem, primitive.ObjectID, error) {
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, [12]byte{}, status.Error(codes.InvalidArgument, err.Error())
	}
	data := &blogItem{}
	filter := bson.M{"_id": objectId}
	log.Println("getting blog with id", objectId)
	res := collection.FindOne(context.Background(), filter)
	if err := res.Decode(data); err != nil {
		return nil, [12]byte{}, status.Errorf(
			codes.NotFound,
			"cannot find blog with id",
			objectId,
		)
	}
	return data, objectId, nil
}

func (*server) ReadBlog(ctx context.Context, req *pb.ReadBlogRequest) (*pb.ReadBlogResponse, error) {
	data, _, err := findById(req.GetId())
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
	data, objectId, err := findById(blog.GetId())
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

func (*server) DeleteBlog(ctx context.Context, req *pb.DeleteBlogRequest) (*pb.DeleteBlogResponse, error) {
	id := req.GetId()
	_, objectId, err := findById(id)
	if err != nil {
		return nil, err
	}
	log.Println("deleting blog with id", objectId)
	res, err := collection.DeleteOne(context.Background(), bson.M{"_id": objectId})
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"cannot delete object in mongo: %v", err)
	}
	if res.DeletedCount == 0 {
		return nil, status.Errorf(
			codes.NotFound,
			"cannot find blog to delete: %v",
			id,
		)
	}
	return &pb.DeleteBlogResponse{
		Id: id,
	}, nil
}

func (*server) ListBlogs(req *pb.ListBlogsRequest, stream pb.BlogService_ListBlogsServer) error {
	count := req.GetCount()
	if count == 0 {
		// default value if empty
		count = 10
	}
	log.Println("listing up to", count, "blogs")
	findOptions := options.Find()
	findOptions.SetLimit(int64(count))
	cursor, err := collection.Find(context.Background(), bson.D{}, findOptions)
	defer cursor.Close(context.Background())
	if err != nil {
		return status.Errorf(
			codes.Internal,
			"error searching for blogs: %v",
			err,
		)
	}
	data := &blogItem{}
	for cursor.Next(context.Background()) {
		err = cursor.Decode(data)
		if err != nil {
			return status.Errorf(
				codes.Internal,
				"error decoding blog item: %v",
				err,
			)
		}
		err = stream.Send(&pb.ListBlogsResponse{
			Blog: &pb.Blog{
				Id:       data.ID.Hex(),
				AuthorId: data.AuthorId,
				Title:    data.Title,
				Content:  data.Content,
			},
		})
		if err != nil {
			return status.Errorf(
				codes.Internal,
				"error sending data to client: %v",
				err,
			)
		}
	}
	if err := cursor.Err(); err != nil {
		return status.Errorf(
			codes.Internal,
			"Unknown cursor error: %v",
			err,
		)
	}
	return nil
	// var items []*blogItem
	// err = cursor.All(context.Background(), items)
	// if err != nil {
	// 	return status.Errorf(
	// 		codes.Internal,
	// 		"error retrieving blogs: %v",
	// 		err,
	// 	)
	// }
	// for _, item := range items {
	//
	// }
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
	reflection.Register(s)

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
