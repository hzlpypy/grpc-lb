package main

import (
	"flag"
	"fmt"
	etcdv3 "github.com/ozonru/etcd/v3/clientv3"
	"github.com/hzlpypy/grpc-lb/common"
	"github.com/hzlpypy/grpc-lb/examples/proto"
	"github.com/hzlpypy/grpc-lb/registry"
	"github.com/hzlpypy/grpc-lb/registry/etcd3"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var nodeID = flag.String("node", "node1", "node ID")
var port = flag.Int("port", 8080, "listening port")

type RpcServer struct {
	addr string
	s    *grpc.Server
}

func NewRpcServer(addr string) *RpcServer {
	s := grpc.NewServer()
	rs := &RpcServer{
		addr: addr,
		s:    s,
	}
	return rs
}

func (s *RpcServer) Run() {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return
	}
	log.Printf("rpc listening on:%s", s.addr)

	proto.RegisterTestServer(s.s, s)
	s.s.Serve(listener)
}

func (s *RpcServer) Stop() {
	s.s.GracefulStop()
}

func (s *RpcServer) Say(ctx context.Context, req *proto.SayReq) (*proto.SayResp, error) {
	text := "Hello " + req.Content + ", I am " + *nodeID
	log.Println(text)

	return &proto.SayResp{Content: text}, nil
}

func StartService() {
	etcdConfg := etcdv3.Config{
		Endpoints: []string{"http://127.0.0.1:2379"},
	}

	service := &registry.ServiceInfo{
		InstanceId: *nodeID,
		Name:       "test",
		Version:    "1.0",
		Address:    fmt.Sprintf("127.0.0.1:%d", *port),
		Metadata:   metadata.Pairs(common.WeightKey, "1"),
	}

	registrar, err := etcd.NewRegistrar(
		&etcd.Config{
			EtcdConfig:  etcdConfg,
			RegistryDir: "/backend/services",
			Ttl:         10 * time.Second,
		})
	if err != nil {
		log.Panic(err)
		return
	}
	server := NewRpcServer(fmt.Sprintf("0.0.0.0:%d", *port))
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		// start service
		server.Run()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		registrar.Register(service)
		wg.Done()
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	registrar.Unregister(service)
	server.Stop()
	wg.Wait()
}

//go run main.go -node node1 -port 28544
//go run main.go -node node2 -port 18562
//go run main.go -node node3 -port 27772
func main() {
	flag.Parse()
	StartService()
}
