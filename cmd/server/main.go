package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/kpfaulkner/collablite/pkg/server"
	"github.com/kpfaulkner/collablite/pkg/storage"
	"github.com/kpfaulkner/collablite/proto"
	"google.golang.org/grpc"
)

func main() {
	fmt.Printf("So it begins...\n")
	port := flag.Int("port", 50051, "The server port")
	flag.Parse()

	//defer profile.Start(profile.MemProfile, profile.MemProfileRate(1), profile.ProfilePath(".")).Stop()
	//defer profile.Start(profile.CPUProfile, profile.ProfilePath(".")).Stop()

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption

	db, err := storage.NewDBSQLite("collablite.db")
	if err != nil {
		log.Fatalf("failed to create db: %v", err)
	}

	grpcServer := grpc.NewServer(opts...)
	proto.RegisterCollabLiteServer(grpcServer, server.NewCollabLiteServer(db))
	grpcServer.Serve(lis)
}
