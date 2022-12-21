package main

import (
	"flag"
	"fmt"
	"net"

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
	store := flag.Bool("store", false, "store data to disk")
	flag.Parse()

	common.SetLogLevel(*logLevel)

	//defer profile.Start(profile.MemProfile, profile.MemProfileRate(1), profile.ProfilePath(".")).Stop()
	//defer profile.Start(profile.CPUProfile, profile.ProfilePath(".")).Stop()

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption

	var db storage.DB

	if *store {
		//db, err = storage.NewDBSQLite("collablite.db")
		//db, err = storage.NewBadgerDB("badgerdb")
		db, err = storage.NewPebbleDB("pebbledb")
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
