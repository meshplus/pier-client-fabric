package main

import (
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	ConfigName = "mock.toml"
	DirectMode = "direct"
	RelayMode  = "relay"
)

type Config struct {
	Appchain AppchainInfo `toml:"appchain" json:"appchaini"`
	Mode     Mode         `toml:"mode" json:"mode"`
}
type AppchainInfo struct {
	BxhId      string `toml:"bxh_id" json:"bxh_id" mapstructure:"bxh_id"`
	AppchainId string `toml:"appchain_id" json:"appchain_id" mapstructure:"appchain_id"`
	Port       string `toml:"port" json:"port"`
}

type Mode struct {
	Type   string `toml:"type" json:"type"`
	Direct Direct `toml:"direct" json:"direct"`
}

type Direct struct {
	ChainID       string `toml:"chainId" json:"chainId"`
	ServiceID     string `toml:"serviceId" json:"serviceId"`
	TimeoutPeriod int    `toml:"timeout_period" json:"timeout_period" mapstructure:"timeout_period"`
	RuleAddr      string `toml:"rule_addr" json:"rule_addr" mapstructure:"rule_addr"`
}

func DefaultConfig() *Config {
	return &Config{
		Appchain: AppchainInfo{
			BxhId:      "",
			AppchainId: "",
			Port:       "",
		},
		Mode: Mode{
			Type: "",
			Direct: Direct{
				ChainID:       "",
				ServiceID:     "",
				RuleAddr:      "0x00000000000000000000000000000000000000a2",
				TimeoutPeriod: 60,
			},
		},
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
