package etcd

import (
	"encoding/json"
	"fmt"
	"github.com/hzlpypy/grpc-lb/registry"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
	"golang.org/x/net/context"
	"google.golang.org/grpc/grpclog"
	"sync"
	"time"
)

//创建租约注册服务
type ServiceReg struct {
	lease         clientv3.Lease
	leaseResp     *clientv3.LeaseGrantResponse
	canclefunc    func()
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
	key           string
}


type Registrar struct {
	sync.RWMutex
	conf        *Config
	Etcd3Client *clientv3.Client
	canceler    map[string]context.CancelFunc
	serviceReg  ServiceReg
}

type Config struct {
	EtcdConfig  clientv3.Config
	RegistryDir string
	Ttl         time.Duration
}

func NewRegistrar(conf *Config) (*Registrar, error) {
	client, err := clientv3.New(conf.EtcdConfig)
	if err != nil {
		return nil, err
	}

	registry := &Registrar{
		Etcd3Client: client,
		conf:        conf,
		canceler:    make(map[string]context.CancelFunc),
	}
	return registry, nil
}

func (r *Registrar) Register(service *registry.ServiceInfo) error {
	val, err := json.Marshal(service)
	if err != nil {
		return err
	}

	key := r.conf.RegistryDir + "/" + service.Name + "/" + service.Version + "/" + service.InstanceId
	fmt.Println(key+"\n")
	value := string(val)
	ctx, cancel := context.WithCancel(context.Background())
	r.Lock()
	r.canceler[service.InstanceId] = cancel
	r.Unlock()
	ttl := int64(r.conf.Ttl / time.Second)
	insertFunc := func() error {
		// 设置租约时间
		resp, err := r.Etcd3Client.Grant(ctx, ttl)
		if err != nil {
			fmt.Printf("[Register] %v\n", err.Error())
			return err
		}
		_, err = r.Etcd3Client.Get(ctx, key)
		if err != nil {
			if err == rpctypes.ErrKeyNotFound {
				if _, err := r.Etcd3Client.Put(ctx, key, value, clientv3.WithLease(resp.ID)); err != nil {
					grpclog.Infof("grpclb: set key '%s' with ttl to etcd3 failed: %s", key, err.Error())
				}
			} else {
				grpclog.Infof("grpclb: key '%s' connect to etcd3 failed: %s", key, err.Error())
			}
			return err
		} else {
			// refresh set to true for not notifying the watcher
			if _, err := r.Etcd3Client.Put(ctx, key, value, clientv3.WithLease(resp.ID)); err != nil {
				grpclog.Infof("grpclb: refresh key '%s' with ttl to etcd3 failed: %s", key, err.Error())
				return err
			}
		}
		if err := r.setLease(resp); err != nil {
			return  err
		}
		go r.ListenLeaseRespChan()
		return nil
	}

	err = insertFunc()
	if err != nil {
		return err
	}
	// Deprecated.  Will be removed in a future 1.x release.
	//ticker := time.NewTicker(r.conf.Ttl / 5)
	//for {
	//	select {
	//	case <-ticker.C:
	//		insertFunc()
	//	case <-ctx.Done():
	//		ticker.Stop()
	//		if _, err := r.Etcd3Client.Delete(context.Background(), key); err != nil {
	//			grpclog.Infof("grpclb: deregister '%s' failed: %s", key, err.Error())
	//		}
	//		return nil
	//	}
	//}

	return nil
}

//设置租约
func (r *Registrar) setLease(leaseResp *clientv3.LeaseGrantResponse) error {
	//lease := clientv3.NewLease(this.Etcd3Client)

	//设置租约时间
	//leaseResp, err := this.Etcd3Client.Grant(context.TODO(), timeNum)
	//if err != nil {
	//	return err
	//}

	//设置续租
	ctx, cancelFunc := context.WithCancel(context.TODO())
	leaseRespChan, err := r.Etcd3Client.KeepAlive(ctx, leaseResp.ID)

	if err != nil {
		return err
	}

	r.serviceReg.lease = r.Etcd3Client
	r.serviceReg.leaseResp = leaseResp
	r.serviceReg.canclefunc = cancelFunc
	r.serviceReg.keepAliveChan = leaseRespChan
	return nil
}

//监听 续租情况
func (this *Registrar) ListenLeaseRespChan() {
	for {
		select {
		case leaseKeepResp := <-this.serviceReg.keepAliveChan:
			if leaseKeepResp == nil {
				fmt.Printf("已经关闭续租功能\n")
				return
			} else {
				fmt.Printf("续租成功\n")
			}
		}
	}
}

func (r *Registrar) Unregister(service *registry.ServiceInfo) error {
	r.RLock()
	cancel, ok := r.canceler[service.InstanceId]
	r.RUnlock()

	if ok {
		cancel()
	}
	return nil
}

func (r *Registrar) Close() {
	r.Etcd3Client.Close()
}
