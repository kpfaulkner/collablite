package main

import (
	"flag"
	"fmt"
	"net"
	"path"

	"github.com/kpfaulkner/collablite/cmd/common"
	log "github.com/sirupsen/logrus"

	"github.com/kpfaulkner/collablite/pkg/server"
	"github.com/kpfaulkner/collablite/pkg/storage"
	"github.com/kpfaulkner/collablite/proto"
	"google.golang.org/grpc"
)

func main() {
	fmt.Printf("So it begins...\n")
	port := flag.Int("port", 50051, "The server port")
	logLevel := flag.String("loglevel", "info", "Log Level: debug, info, warn, error")
	store := flag.Bool("store", false, "Store data to disk")
	storePath := flag.String("storepath", ".", "Path of storage location (if persist to local disk)")

	flag.Parse()

	common.SetLogLevel(*logLevel)

	//defer profile.Start(profile.MemProfile, profile.MemProfileRate(1), profile.ProfilePath(".")).Stop()
	//defer profile.Start(profile.CPUProfile, profile.ProfilePath(".")).Stop()
	//defer profile.Start(profile.TraceProfile, profile.ProfilePath(".")).Stop()
	//defer profile.Start(profile.MutexProfile, profile.ProfilePath(".")).Stop()
	//defer profile.Start(profile.GoroutineProfile, profile.ProfilePath(".")).Stop()

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption

	var db storage.DB

	if *store {
		pebbleClient, err := storage.NewPebbleClient(path.Join(*storePath, "pebbledb"))
		if err != nil {
			log.Fatalf("unable to create pebble client: %v", err)
		}
		db, err = storage.NewPebbleDB(pebbleClient)
		if err != nil {
			log.Fatalf("failed to create db: %v", err)
		}
	} else {
		db, err = storage.NewNullDB()
		if err != nil {
			log.Fatalf("failed to create nulldb: %v", err)
		}
	}

	grpcServer := grpc.NewServer(opts...)
	proto.RegisterCollabLiteServer(grpcServer, server.NewCollabLiteServer(db))
	grpcServer.Serve(lis)
}
