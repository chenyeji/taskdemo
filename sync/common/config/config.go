package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	App       `toml:"app"`
	Log       `toml:"log"`
	Producers map[string]*Producer `toml:"producer"`
	Consumers map[string]*Consumer `toml:"consumer"`
}

type App struct {
	Name          string   `toml:"name"`
	Chains        []string `toml:"chain"`
	EmptyInterval int      `toml:"empty_interval"`
	ErrorInterval int      `toml:"error_interval"`
}

type Log struct {
	Stdout `toml:"stdout"`
	File   `toml:"file"`
}

type Stdout struct {
	Enable bool `toml:"enable"`
	Level  int  `toml:"level"`
}

type File struct {
	Enable bool   `toml:"enable"`
	Level  int    `toml:"level"`
	Path   string `toml:"path"`
	MaxAge int    `toml:"max_age"`
}

type Producer struct {
	URL      string `toml:"url"`
	Timeout  int    `toml:"timeout"`
	User     string `toml:"user"`
	Password string `toml:"password"`
}

type Consumer struct {
	StartHeight int `toml:"start_height"`
}

func NewConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := NewConfigFromRawData(string(data))
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewConfigFromRawData(data string) (*Config, error) {
	cfg := &Config{}
	_, err := toml.Decode(data, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func DefaultConfig(app string) *Config {
	return &Config{
		App: App{
			Name: app,
		},
		Log: Log{
			Stdout: Stdout{
				Enable: true,
				Level:  5,
			},
			File: File{
				Enable: true,
				Level:  5,
				MaxAge: 7,
				Path:   fmt.Sprintf("./logs/%s/%s.log", app, app),
			},
		},
	}
}
