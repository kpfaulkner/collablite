package main

import (
	"flag"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/kpfaulkner/collablite/pkg/server"
	"github.com/kpfaulkner/collablite/pkg/storage"
	"github.com/kpfaulkner/collablite/proto"
	"google.golang.org/grpc"
)

func setLogLevel(ll string) {
	switch ll {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}

func main() {
	fmt.Printf("So it begins...\n")
	port := flag.Int("port", 50051, "The server port")
	logLevel := flag.String("loglevel", "info", "Log Level: debug, info, warn, error")
	flag.Parse()

	setLogLevel(*logLevel)

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
