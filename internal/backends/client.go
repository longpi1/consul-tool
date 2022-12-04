package backends

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	errors "github.com/longpi1/consul-tool/pkg/error"
	"github.com/longpi1/consul-tool/pkg/log"
	"strings"
	"sync"
	"time"
)

// Option ...
type Option func(opt *Config)

// NewConfig 初始化consul配置
func NewConfig(opts ...Option) *Config {
	c := &Config{
		conf:     api.DefaultConfig(),
		watchers: make(map[string]*watcher),
		logger:   log.NewLogger(),
	}

	for _, o := range opts {
		o(c)
	}

	return c
}

// Config ...
type Config struct {
	sync.RWMutex
	logger   log.Logger
	kv       *api.KV
	conf     *api.Config
	watchers map[string]*watcher
	prefix   string
}

// CheckWatcher ...
func (c *Config) CheckWatcher(path string) error {
	c.RLock()
	defer c.RUnlock()

	if _, ok := c.watchers[c.absPath(path)]; ok {
		return errors.ErrAlreadyWatch
	}

	return nil
}

func (c *Config) getWatcher(path string) *watcher {
	c.RLock()
	defer c.RUnlock()

	return c.watchers[c.absPath(path)]
}

// 添加watcher
func (c *Config) addWatcher(path string, w *watcher) error {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.watchers[c.absPath(path)]; ok {
		return errors.ErrAlreadyWatch
	}

	c.watchers[c.absPath(path)] = w
	return nil
}

func (c *Config) removeWatcher(path string) {
	c.Lock()
	defer c.Unlock()

	delete(c.watchers, c.absPath(path))
}

func (c *Config) cleanWatcher() {
	c.Lock()
	defer c.Unlock()

	for k, w := range c.watchers {
		w.stop()
		delete(c.watchers, k)
	}
}

// 获取所有的watcher
func (c *Config) getAllWatchers() []*watcher {
	c.RLock()
	defer c.RUnlock()

	watchers := make([]*watcher, 0, len(c.watchers))
	for _, w := range c.watchers {
		watchers = append(watchers, w)
	}

	return watchers
}

func (c *Config) watcherLoop(path string) {
	c.logger.Info("watcher start...", "path", path)

	w := c.getWatcher(path)
	if w == nil {
		c.logger.Error("watcher not found", "path", path)
		return
	}

	for {
		if err := w.run(c.conf.Address, c.conf); err != nil {
			c.logger.Warn("watcher connect error", "path", path, "error", err)
			time.Sleep(time.Second * 3)
		}

		w = c.getWatcher(path)
		if w == nil {
			c.logger.Info("watcher stop", "path", path)
			return
		}

		c.logger.Warn("watcher reconnect...", "path", path)
	}
}

// 重置consul的watcher
func (c *Config) Reset() error {
	watchMap := c.getAllWatchers()

	for _, w := range watchMap {
		w.stop()
	}

	return c.Init()
}

// Init 初始化consul客户端
func (c *Config) Init() error {
	client, err := api.NewClient(c.conf)
	if err != nil {
		return fmt.Errorf("init fail: %w", err)
	}

	c.kv = client.KV()
	return nil
}

// Put 插入该路径的kv
func (c *Config) Put(path string, value interface{}) error {
	var (
		data []byte
		err  error
	)

	data, err = json.Marshal(value)
	if err != nil {
		data = []byte(fmt.Sprintf("%v", value))
	}

	p := &api.KVPair{Key: c.absPath(path), Value: data}
	_, err = c.kv.Put(p, nil)
	if err != nil {
		return fmt.Errorf("put fail: %w", err)
	}
	return nil
}

// Get 获取该路径的kv
func (c *Config) Get(keys ...string) (ret *KV) {
	var (
		path   = c.absPath(keys...) + "/"
		fields []string
	)

	ret = &KV{}
	ks, err := c.list()
	if err != nil {
		ret.err = fmt.Errorf("get list fail: %w", err)
		return
	}

	for _, k := range ks {
		if !strings.HasPrefix(path, k+"/") {
			ret.err = errors.ErrKeyNotFound
			continue
		}
		field := strings.TrimSuffix(strings.TrimPrefix(path, k+"/"), "/")
		if len(field) != 0 {
			fields = strings.Split(field, "/")
		}

		kvPair, _, err := c.kv.Get(k, nil)
		ret.value = kvPair.Value
		ret.key = strings.TrimSuffix(strings.TrimPrefix(path, c.prefix+"/"), "/")
		if err != nil {
			err = fmt.Errorf("get fail: %w", err)
		}
		ret.err = err
		break
	}

	if len(fields) == 0 {
		return
	}
	ret.key += "/" + strings.Join(fields, "/")
	return
}

// Delete 删除该路径的kv
func (c *Config) Delete(path string) error {
	_, err := c.kv.Delete(c.absPath(path), nil)
	if err != nil {
		return fmt.Errorf("delete fail: %w", err)
	}
	return nil
}

// Watch   实现监听
func (c *Config) Watch(path string, handler func(*KV)) error {
	// 初始化watcher
	watcher, err := newWatcher(c.absPath(path))
	if err != nil {
		return fmt.Errorf("watch fail: %w", err)
	}
	// 对应的路径发生变化时，调用对应的处理函数
	watcher.setHybridHandler(c.prefix, handler)
	// 相应路径下添加对应的wathcer用于实现watch机制
	err = c.addWatcher(path, watcher)
	if err != nil {
		return err
	}
	// 调用协程循环监听
	go c.watcherLoop(path)
	return nil
}

// StopWatch 停止监听
func (c *Config) StopWatch(path ...string) {
	if len(path) == 0 {
		c.cleanWatcher()
		return
	}

	for _, p := range path {
		wp := c.getWatcher(p)
		if wp == nil {
			c.logger.Info("watcher already stop", "path", p)
			continue
		}

		c.removeWatcher(p)
		wp.stop()
		for !wp.IsStopped() {
		}
	}
}

// 获取绝对路径
func (c *Config) absPath(keys ...string) string {
	if len(keys) == 0 {
		return c.prefix
	}

	if len(keys[0]) == 0 {
		return c.prefix
	}

	if len(c.prefix) == 0 {
		return strings.Join(keys, "/")
	}

	return c.prefix + "/" + strings.Join(keys, "/")
}

func (c *Config) list() ([]string, error) {
	keyPairs, _, err := c.kv.List(c.prefix, nil)
	if err != nil {
		return nil, err
	}

	list := make([]string, 0, len(keyPairs))
	for _, v := range keyPairs {
		if len(v.Value) != 0 {
			list = append(list, v.Key)
		}
	}

	return list, nil
}

// WithPrefix ...
func WithPrefix(prefix string) Option {
	return func(c *Config) {
		c.prefix = prefix
	}
}

// WithAddress ...
func WithAddress(address string) Option {
	return func(c *Config) {
		c.conf.Address = address
	}
}

// Withlogger ...
func Withlogger(logger log.Logger) Option {
	return func(c *Config) {
		c.logger = logger
	}
}
