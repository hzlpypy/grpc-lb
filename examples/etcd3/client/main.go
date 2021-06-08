package main

import (
	etcd3 "go.etcd.io/etcd/clientv3"
	"github.com/hzlpypy/grpc-lb/balancer"
	"github.com/hzlpypy/grpc-lb/examples/proto"
	registry "github.com/hzlpypy/grpc-lb/registry/etcd3"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"time"
)

func main() {
	etcdConfg := etcd3.Config{
		Endpoints: []string{"http://127.0.0.1:2379"},
	}
	registry.RegisterResolver("etcd3", etcdConfg, "/backend/services", "test", "1.0")

	c, err := grpc.Dial("etcd3:///", grpc.WithInsecure(), grpc.WithBalancerName(balancer.RoundRobin))
	//c, err := grpc.Dial("etcd3:///", grpc.WithInsecure())
	if err != nil {
		log.Printf("grpc dial: %s", err)
		return
	}
	defer c.Close()
	client := proto.NewTestClient(c)

	for i := 0; i < 500; i++ {
		resp, err := client.Say(context.Background(), &proto.SayReq{Content: "round robin"})
		if err != nil {
			log.Println("aa:", err)
			time.Sleep(time.Second)
			continue
		}
		time.Sleep(time.Second)
		log.Printf(resp.Content)
	}
}
