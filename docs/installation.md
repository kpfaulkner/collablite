# Installation


## Requirements

- Go (using 1.19 for development)
- Protobuf compiler (protoc)

To run the server:

```

protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative .\proto\collablite.proto

cd cmd/server
go build .
./server
```

OR

```
buildserver.cmd

or

buildserver.sh
```

By default it will be listening on port 50511 (gRPC) and will create a Pebble DB directory cmd/server/pebble for persistent storage.


