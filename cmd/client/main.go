package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/kpfaulkner/collablite/client"
	"github.com/kpfaulkner/collablite/cmd/common"
	log "github.com/sirupsen/logrus"
)

func processObjectConfirmation(obj *client.ChangeConfirmation) error {
	//log.Debugf("confirmation: %v", obj)
	return nil
}

func main() {
	fmt.Printf("So it begins...\n")
	host := flag.String("host", "localhost:50051", "host:port of server")
	id := flag.String("id", "ken1", "id of client")
	objectID := flag.String("objectid", "testobject1", "objectid of object to write/watch")
	send := flag.Bool("send", false, "send data to server")
	logLevel := flag.String("loglevel", "info", "Log Level: debug, info, warn, error")
	flag.Parse()

	common.SetLogLevel(*logLevel)

	cli := client.NewClient(*host)

	wg := sync.WaitGroup{}
	wg.Add(1)

	ctx := context.Background()
	cli.RegisterCallback(processObjectConfirmation)
	cli.Connect(ctx)
	go cli.Listen(ctx)

	if *send {
		go func() {
			for i := 0; i < 1000000000; i++ {

				req := client.OutgoingChange{
					ObjectID:   *objectID,
					PropertyID: fmt.Sprintf("property-%03d", rand.Intn(100)),
					Data:       []byte(fmt.Sprintf("hello world-%s-%d", *id, i)),
				}
				if err := cli.SendChange(&req); err != nil {
					log.Errorf("failed to send change: %v", err)
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	cli.RegisterToObject(nil, *objectID)

	wg.Wait()
}
