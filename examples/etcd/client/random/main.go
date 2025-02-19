package main

import (
	etcd "github.com/ozonru/etcd/v3/client"
	"github.com/hzlpypy/grpc-lb/balancer"
	"github.com/hzlpypy/grpc-lb/examples/proto"
	registry "github.com/hzlpypy/grpc-lb/registry/etcd"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"time"
)

func main() {
	etcdConfg := etcd.Config{
		Endpoints: []string{"http://10.0.101.68:2379"},
	}
	registry.RegisterResolver("etcd", etcdConfg, "/backend/services", "test", "1.0")

	c, err := grpc.Dial("etcd:///", grpc.WithInsecure(), grpc.WithBalancerName(balancer.Random))
	if err != nil {
		log.Printf("grpc dial: %s", err)
		return
	}
	defer c.Close()

	client := proto.NewTestClient(c)
	for i := 0; i < 500; i++ {

		resp, err := client.Say(context.Background(), &proto.SayReq{Content: "round robin"})
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second)
			continue
		}
		time.Sleep(time.Second)
		log.Printf(resp.Content)
	}
}
