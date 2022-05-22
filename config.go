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
	BxhId      string `toml:"bxh_id" json:"bxh_id" mapstructure:"bxh_id"`
	AppchainId string `toml:"appchain_id" json:"appchain_id" mapstructure:"appchain_id"`
	Port       string `toml:"port" json:"port"`
}

type Service struct {
	ID   string `toml:"id" json:"id"`
	Name string `toml:"name" json:"name"`
	Type string `toml:"type" json:"type"`
}

func DefaultConfig() *Config {
	return &Config{
		Fabric: Fabric{
			BxhId:      "",
			AppchainId: "",
			Port:       "",
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
