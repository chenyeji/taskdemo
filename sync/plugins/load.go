package plugins

import (
	"fmt"

	"gitlab.com/sync/plugins/btc"

	"github.com/pkg/errors"

	"gitlab.com/sync/common/config"
	"gitlab.com/sync/features"
	"gitlab.com/sync/plugins/eth"
)

type Plugin struct {
	Producer features.Producer
	Consumer features.Consumer
}

type newProducer func(cfg *config.Producer) (features.Producer, error)
type newConsumer func(cfg *config.Consumer) (features.Consumer, error)

var (
	supportedProducer = map[string]newProducer{
		"btc": btc.NewProducer,
		"eth": eth.NewProducer,
	}
	supportedConsumer = map[string]newConsumer{
		"btc": btc.NewConsumer,
		"eth": eth.NewConsumer,
	}
)

func Loader(chains []string, cfg *config.Config) (map[string]*Plugin, error) {
	result := make(map[string]*Plugin)
	for _, v := range chains {
		p := &Plugin{}
		if f, ok := supportedProducer[v]; !ok {
			return nil, fmt.Errorf("unsupported chain %s", v)
		} else if producer, err := f(cfg.Producers[v]); err != nil {
			return nil, errors.Wrapf(err, "init chain %s", v)
		} else {
			p.Producer = producer
		}
		if f, ok := supportedConsumer[v]; !ok {
			return nil, fmt.Errorf("unsupported chain %s", v)
		} else if consumer, err := f(cfg.Consumers[v]); err != nil {
			return nil, errors.Wrapf(err, "init chain %s", v)
		} else {
			p.Consumer = consumer
		}
		result[v] = p
	}
	return result, nil
}
