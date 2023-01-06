# Installation

To run the server:

```
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


