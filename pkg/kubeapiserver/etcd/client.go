package etcd

// kube-apiserver的etcd存储客户端
import (
	"context"
	"log"
	"time"

	cliv3 "go.etcd.io/etcd/client/v3"
)

// etcdcli 配置
type Config struct {
	// etcd地址
	Endpoints []string
	// 超时时间
	DialTimeout time.Duration
}

// 默认配置
var defaultCfg = Config{
	Endpoints:   []string{"localhost:2379"},
	DialTimeout: 3 * time.Second,
}

// etcd 存储接口
type Store interface {
	// 获取key的值
	Get(key string) (string, error)
	// 设置key的值
	Set(key, value string) error
	// 删除key
	Delete(key string) error

	// TODO: cascade 操作

	// // 获取key的子key
	// GetSubKeys(key string) ([]string, error)
	// // 获取key的子key的值
	GetSubKeysValues(key string) (map[string]string, error)
	// // 设置key的子key的值
	// SetSubKeysValues(key string, values map[string]string) error
	// // 删除key的子key
	DeleteSubKeys(key string) error

	// TODO: watch 连接
}
type store struct {
	// etcd配置
	cfg Config
	// etcd客户端
	cli *cliv3.Client
}

func NewEtcdStore() (*store, error) {

	var store store

	err := store.init(defaultCfg)

	return &store, err
}

// 初始化etcd存储
func (s *store) init(cfg Config) error {
	log.Println("etcd store init")
	// 创建etcd客户端
	cli, err := cliv3.New(cliv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: s.cfg.DialTimeout,
	})
	if err != nil {
		return err
	}
	s.cli = cli
	return nil
}

// 获取key的值
func (s *store) Get(key string) (string, error) {
	log.Println("get key in store", key)
	kv := cliv3.NewKV(s.cli)
	res, err := kv.Get(context.TODO(), key)

	if err != nil {
		return "", err
	}
	// 返回key的值
	if res.Count > 0 {
		return string(res.Kvs[0].Value), nil
	}

	return "", nil
}

// 设置key的值
func (s *store) Set(key, value string) error {
	log.Println("set key in store", key, value)
	kv := cliv3.NewKV(s.cli)
	_, err := kv.Put(context.TODO(), key, value)
	if err != nil {
		return err
	}

	return nil
}

// 删除key
func (s *store) Delete(key string) error {
	log.Println("delete key in store", key)
	kv := cliv3.NewKV(s.cli)
	_, err := kv.Delete(context.TODO(), key)
	if err != nil {
		return err
	}

	return nil
}

func (s *store) GetSubKeysValues(key string) (map[string]string, error) {
	log.Println("get subkeys values in store", key)
	kv := cliv3.NewKV(s.cli)
	res, err := kv.Get(context.TODO(), key, cliv3.WithPrefix())

	if err != nil {
		return nil, err
	}

	values := make(map[string]string)
	for _, kv := range res.Kvs {
		values[string(kv.Key)] = string(kv.Value)
	}

	return values, nil

}

func (s *store) DeleteSubKeys(key string) error {
	kv := cliv3.NewKV(s.cli)
	_, err := kv.Delete(context.TODO(), key, cliv3.WithPrefix())
	if err != nil {
		return err
	}

	return nil
}
