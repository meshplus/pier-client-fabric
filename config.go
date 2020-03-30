package main

import (
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	ConfigName = "fabric.toml"
)

type Fabric struct {
	Addr        string `toml:"addr" json:"addr"`
	Name        string `toml:"name" json:"name"`
	EventFilter string `mapstructure:"event_filter" toml:"event_filter" json:"event_filter"`
	Username    string `toml:"username" json:"username"`
	CCID        string `toml:"ccid" json:"ccid"`
	ChannelId   string `mapstructure:"channel_id" toml:"channel_id" json:"channel_id"`
	Org         string `toml:"org" json:"org"`
}

func DefaultConfig() *Fabric {
	return &Fabric{
		Addr:        "40.125.164.122:10053",
		Name:        "fabric",
		EventFilter: "CrosschainEventName",
		Username:    "Admin",
		CCID:        "Broker-001",
		ChannelId:   "mychannel",
		Org:         "org2",
	}
}

func UnmarshalConfig(configPath string) (*Fabric, error) {
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
