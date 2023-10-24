package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/sync/core"

	"gitlab.com/sync/common"

	"github.com/sirupsen/logrus"

	"gitlab.com/sync/common/config"
	"gitlab.com/sync/common/log"
)

var (
	configFile = flag.String("config", "config.toml", "set the config file path")
)

// curl -i -X POST -H 'content-type: application/json' --data '{"jsonrpc":"2.0", "method":"eth_blockNumber","params":[],"id":1}' https://rpc.ankr.com/eth_goerli/8b4a7aff54ac22cd3d15d0e58b3ba1a6ee3f90b2233cba73bd7093dbcfe885dd
// curl -i -X POST -H 'content-type: application/json' --data '{"jsonrpc":"2.0", "method":"eth_getBlockByNumber","params":["0x975417",true],"id":1}' https://rpc.ankr.com/eth_goerli/8b4a7aff54ac22cd3d15d0e58b3ba1a6ee3f90b2233cba73bd7093dbcfe885dd
// btc
// curl -i -X POST  -H 'content-type: application/json' --data '{"jsonrpc":"2.0","method":"getblockcount","params":null,"id":1}' https://maximum-restless-river.btc.quiknode.pro/bcf68d1b628602a9ad4b25f8e1b6cebcc3c686c2/
// curl -i -X POST  -H 'content-type: application/json' --data '{"jsonrpc":"2.0","method":"getblockhash","params":[813469],"id":1}' https://maximum-restless-river.btc.quiknode.pro/bcf68d1b628602a9ad4b25f8e1b6cebcc3c686c2/
// getblock 查询这个块的所有信息
// curl -i -X POST  -H 'content-type: application/json' --data '{"jsonrpc":"2.0","method":"getblock","params":["000000000000000000048a0f02d84fcad61b9d75b50ec01b1bbb2c748129d5b6"],"id":1}' https://maximum-restless-river.btc.quiknode.pro/bcf68d1b628602a9ad4b25f8e1b6cebcc3c686c2/

func main() {
	defer common.Recover()
	flag.Parse()
	c, err := config.NewConfigFromFile(*configFile)
	if err != nil {
		logrus.Fatal(err)
	}
	log.SetLogDetailsByConfig(c)

	shutdown := make(chan struct{})
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		for sig := range signals {
			if sig == os.Interrupt || sig == syscall.SIGTERM {
				logrus.Infof("received signal [%v], preparing to quit", sig)
				close(shutdown)
			} else if sig == syscall.SIGHUP {
				logrus.Infof("received signal [%v], ignored", sig)

				// 先忽略掉该信号
			}
		}
	}()

	if processor, err := core.NewProcessor(c); err != nil {
		logrus.Fatal(err)
	} else {
		processor.Loop(shutdown)
	}
}
