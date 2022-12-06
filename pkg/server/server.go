package server

import (
	pb ""

	"github.com/kpfaulkner/collablite/pkg/storage"
)

// CollabLiteServer receives gRPC requests from clients and modifies the
// object/data accordingly.
type CollabLiteServer struct {
	pb.UnimplementedGreeterServer
	db storage.DB
}

func NewCollabLiteServer(db storage.DB) *CollabLiteServer {
	cls := CollabLiteServer{}
	cls.db = db
	return &cls
}
