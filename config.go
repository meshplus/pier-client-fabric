package main

import (
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	ConfigName = "fabric.toml"
)

type Config struct {
	Fabric   Fabric    `toml:"fabric" json:"fabric"`
	Services []Service `mapstructure:"services" json:"services"`
}
type Fabric struct {
	Addr          string `toml:"addr" json:"addr"`
	Name          string `toml:"name" json:"name"`
	EventFilter   string `mapstructure:"event_filter" toml:"event_filter" json:"event_filter"`
	Username      string `toml:"username" json:"username"`
	CCID          string `toml:"ccid" json:"ccid"`
	ChannelId     string `mapstructure:"channel_id" toml:"channel_id" json:"channel_id"`
	Org           string `toml:"org" json:"org"`
	TimeoutHeight int64  `mapstructure:"timeout_height" json:"timeout_height"`
	ChainID       string `mapstructure:"chain_id" json:"chain_id"`
}

type Service struct {
	ID   string `toml:"id" json:"id"`
	Name string `toml:"name" json:"name"`
	Type string `toml:"type" json:"type"`
}

func DefaultConfig() *Config {
	return &Config{
		Fabric: Fabric{
			Addr:        "40.125.164.122:10053",
			Name:        "fabric",
			EventFilter: "CrosschainEventName",
			Username:    "Admin",
			CCID:        "Broker-001",
			ChannelId:   "mychannel",
			Org:         "org2",
		},
		Services: nil,
	}
}

func UnmarshalConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(filepath.Join(configPath, ConfigName))
	viper.SetConfigType("toml")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("FABRIC")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	config := DefaultConfig()

	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}
