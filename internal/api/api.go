package api

import "github.com/longpi1/consul-tool/internal/backends"

var kvConf = backends.NewConfig()

// Init ...
func Init(opts ...backends.Option) error {
	for _, o := range opts {
		o(kvConf)
	}

	return kvConf.Init()
}

// SetOptions ...
func SetOptions(opts ...backends.Option) error {
	for _, o := range opts {
		o(kvConf)
	}

	return kvConf.Reset()
}

// Put ...
func Put(path string, value interface{}) error {
	return kvConf.Put(path, value)
}

// Delete ...
func Delete(path string) error {
	return kvConf.Delete(path)
}

// Get ...
func Get(keys ...string) *backends.KV {
	return kvConf.Get(keys...)
}

// Watch ...
func Watch(path string, handler func(*backends.KV)) error {
	return kvConf.Watch(path, handler)
}

// StopWatch ...
func StopWatch(path ...string) {
	kvConf.StopWatch(path...)
}
