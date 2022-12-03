package backends

// KV ...
type KV struct {
	err   error
	key   string
	value []byte
}

// Err ...
func (kv *KV) Err() error {
	return kv.err
}

// Key ...
func (kv *KV) Key() string {
	return kv.key
}

// Value ...
func (kv *KV) Value() []byte {
	return kv.value
}
