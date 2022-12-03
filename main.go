package main

import (
	"github.com/longpi1/consul-tool/internal/backends"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 初始化consul配置信息
	cli := backends.NewConfig(backends.WithPrefix("kvTest"))
	if err := cli.Init(); err != nil {
		log.Fatalln(err)
	}
	//监听consul中的key： test
	err := cli.Watch("test", func(r *backends.KV) {
		log.Printf("该key： %s 已经更新", r.Key())
	})
	if err != nil {
		log.Fatalln(err)
	}
	//插入key
	if err := cli.Put("test", "value"); err != nil {
		log.Fatalln(err)
	}
	//读取key
	if ret := cli.Get("test"); ret.Err() != nil {
		log.Fatalln(ret.Err())
	} else {
		println(ret.Value())
	}

	c := make(chan os.Signal, 1)
	// 监听退出相关的syscall
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-c
		log.Printf("exit with signal %s", s.String())
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			//停止监听对应的路径
			cli.StopWatch("test")
			time.Sleep(time.Second * 2)
			close(c)
			return
		case syscall.SIGHUP:
		default:
			close(c)
			return
		}
	}
}
