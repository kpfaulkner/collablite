package server

import (
	"github.com/kpfaulkner/collablite/pkg/storage"
	"github.com/kpfaulkner/collablite/proto"
)

// CollabLiteServer receives gRPC requests from clients and modifies the
// object/data accordingly.
type CollabLiteServer struct {
	proto.UnimplementedCollabLiteServer
	db storage.DB
}

func NewCollabLiteServer(db storage.DB) *CollabLiteServer {
	cls := CollabLiteServer{}
	cls.db = db
	return &cls
}
