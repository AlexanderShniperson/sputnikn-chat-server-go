package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"chatserver/data"
	"chatserver/server"

	pb "chatserver/contract/v1"
	db "chatserver/db"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	tls      = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile = flag.String("cert_file", "", "The TLS cert file")
	keyFile  = flag.String("key_file", "", "The TLS key file")
	port     = flag.Int("port", 50051, "The server port")
	dbUrl    = flag.String("db_url", "postgres://postgres:ok@localhost:5432/sputniknchat", "The DB url connection string")
)

func main() {
	flag.Parse()

	database := db.SetupDatabase(*dbUrl)
	defer database.Close()

	roomManager := server.NewRoomManager(database)
	go roomManager.Start()

	// 30 days duration valid token
	tokenValidDuration := time.Duration(time.Hour * 24 * 30)
	tokenManager := server.NewJWTManager("TheSecret", tokenValidDuration)
	authInterceptor := server.NewAuthInterceptor(tokenManager)

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var opts []grpc.ServerOption
	if *tls {
		if *certFile == "" {
			*certFile = data.Path("x509/server_cert.pem")
		}
		if *keyFile == "" {
			*keyFile = data.Path("x509/server_key.pem")
		}
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials: %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	opts = append(opts, grpc.UnaryInterceptor(authInterceptor.Unary()))
	opts = append(opts, grpc.StreamInterceptor(authInterceptor.Stream()))

	grpcServer := grpc.NewServer(opts...)
	chatService := server.NewChatService(database, tokenManager, roomManager)
	chatStreamService := server.NewChatStreamService(roomManager)
	pb.RegisterChatServiceServer(grpcServer, chatService)
	pb.RegisterChatStreamServiceServer(grpcServer, chatStreamService)
	grpcServer.Serve(lis)
}
