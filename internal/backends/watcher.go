package backends

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
)


type watcher struct {
	sync.RWMutex   // 基于读写锁实现并发控制
	*watch.Plan
	lastValues    map[string][]byte
	hybridHandler watch.HybridHandlerFunc   // 当对应的key发生变化时，调用对应的处理函数
	stopChan      chan struct{}
	err           chan error
}

//初始化对应的watcher ，这里设置的是监听路径的类型，也可以支持service、node等，通过更改type
//支持的type类型有
//key - Watch a specific KV pair
//keyprefix - Watch a prefix in the KV store
//services - Watch the list of available services
//nodes - Watch the list of nodes
//service- Watch the instances of a service
//checks - Watch the value of health checks
//event - Watch for custom user events
func newWatcher(path string) (*watcher, error) {
	wp, err := watch.Parse(map[string]interface{}{"type": "keyprefix", "prefix": path})
	if err != nil {
		return nil, err
	}

	return &watcher{
		Plan:       wp,
		lastValues: make(map[string][]byte),
		err:        make(chan error, 1),
	}, nil
}

func newServiceWatcher(serviceName string) (*watcher, error) {
	wp, err := watch.Parse(map[string]interface{}{"type": "service", "service": serviceName})
	if err != nil {
		return nil, err
	}
	return &watcher{
		Plan:       wp,
		lastValues: make(map[string][]byte),
		err:        make(chan error, 1),
	}, nil
}



//获取value
func (w *watcher) getValue(path string) []byte {
	w.RLock()
	defer w.RUnlock()

	return w.lastValues[path]
}

//更新value
func (w *watcher) updateValue(path string, value []byte) {
	w.Lock()
	defer w.Unlock()

	if len(value) == 0 {
		delete(w.lastValues, path)
	} else {
		w.lastValues[path] = value
	}
}

//用于设置对应的处理函数
func (w *watcher) setHybridHandler(prefix string, handler func(*KV)) {
	w.hybridHandler = func(bp watch.BlockingParamVal, data interface{}) {
		kvPairs := data.(api.KVPairs)
		ret := &KV{}

		for _, k := range kvPairs {
			path := strings.TrimSuffix(strings.TrimPrefix(k.Key, prefix+"/"), "/")
			v := w.getValue(path)

			if len(k.Value) == 0 && len(v) == 0 {
				continue
			}

			if bytes.Equal(k.Value, v) {
				continue
			}

			ret.value = k.Value
			ret.key = path
			w.updateValue(path, k.Value)
			handler(ret)
		}
	}
}

func (w *watcher) run(address string, conf *api.Config) error {
	w.stopChan = make(chan struct{})
	w.Plan.HybridHandler = w.hybridHandler

	go func() {
		w.err <- w.RunWithConfig(address, conf)
	}()

	select {
	case err := <-w.err:
		return fmt.Errorf("run fail: %w", err)
	case <-w.stopChan:
		w.Stop()
		return nil
	}
}

func (w *watcher) stop() {
	close(w.stopChan)
}
