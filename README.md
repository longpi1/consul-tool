# consul-tool
用于获取consul中键/值以及实现consul Watch机制监听的工具库

### 初始化配置
```golang
conf := NewConfig()
```

### With Options
```golang
conf := NewConfig(
    WithPrefix(prefix),             // consul kv prefix
    WithAddress(address),           // consul address
)

```

### Init
```golang
if err := conf.Init();err !=nil {
    return err
}
```

### Put
```golang
if err := conf.Put(key, value);err !=nil {
    return err
}
```

### Delete
```golang
if err := conf.Delete(key);err !=nil {
    return err
}
```

### Get
```golang
// 获取key
key := conf.KV.Key()

```

### Get
```golang
// 获取value
key := conf.KV.Value()

```

### Watch
```golang
conf.Watch(path, func(r *KV){
    r.Scan(x)
})

```

### Stop Watch
```golang
// stop single watcher
conf.StopWatch(path)

// stop multiple watcher
conf.StopWatch(path1, path2)

// stop all watcher
conf.StopWatch()
```
