package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr"
	"github.com/hashicorp/go-plugin"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/meshplus/pier/pkg/plugins"
	"github.com/urfave/cli"
)

var initCMD = cli.Command{
	Name:  "init",
	Usage: "Get appchain default configuration",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:     "target",
			Usage:    "Specify where to put the default configuration files",
			Required: false,
		},
	},
	Action: func(ctx *cli.Context) error {
		target := ctx.String("target")
		box := packr.NewBox("config")

		if err := box.Walk(func(s string, file packd.File) error {
			p := filepath.Join(target, s)
			dir := filepath.Dir(p)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				err := os.MkdirAll(dir, 0755)
				if err != nil {
					return err
				}
			}
			return ioutil.WriteFile(p, []byte(file.String()), 0644)
		}); err != nil {
			return err
		}

		return nil
	},
}

var getValidator = cli.Command{
	Name:  "validator",
	Usage: "Get fabric validator info",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:     "config",
			Usage:    "Specify config addr",
			Required: true,
		},
	},
	Action: func(ctx *cli.Context) error {
		configPath := ctx.String("config")
		fabconfig, err := UnmarshalConfig(configPath)
		if err != nil {
			return fmt.Errorf("unmarshal config for plugin :%w", err)
		}
		fabricConfig := fabconfig.Fabric
		contractmeta := &ContractMeta{
			Username:  fabricConfig.Username,
			CCID:      fabricConfig.CCID,
			ChannelID: fabricConfig.ChannelId,
			ORG:       fabricConfig.Org,
		}
		configProvider := config.FromFile(filepath.Join(configPath, "config.yaml"))
		sdk, err := fabsdk.New(configProvider)
		if err != nil {
			return fmt.Errorf("create sdk fail: %s\n", err)
		}

		channelProvider := sdk.ChannelContext(contractmeta.ChannelID, fabsdk.WithUser(contractmeta.Username), fabsdk.WithOrg(contractmeta.ORG))

		l, err := ledger.New(channelProvider)
		if err != nil {
			return err
		}
		// Get Fabric Channel Config
		conf, err := l.QueryConfig()
		if err != nil {
			return err
		}
		// Generate Policy Bytes
		policy, err := cauthdsl.FromString("AND('Org2MSP.peer', 'Org1MSP.peer')")
		if err != nil {
			return err
		}
		siz := policy.XXX_Size()
		b := make([]byte, 0, siz)
		pBytes, err := policy.XXX_Marshal(b, false)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile("policy", []byte(string(pBytes)), 777); err != nil {
			return err
		}
		validator := &Validator{
			Cid:     contractmeta.CCID,
			ChainId: "",
			Policy:  string(pBytes),
		}
		// Generate MSP config bytes
		var confStrs []string
		for i, value := range conf.MSPs() {
			confStrs = append(confStrs, value.String())
			err := ioutil.WriteFile("conf"+strconv.Itoa(i), []byte(value.String()), 777)
			if err != nil {
				return err
			}
		}
		validator.ConfByte = confStrs
		validatorBytes, err := json.Marshal(validator)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile("validator", validatorBytes, 777); err != nil {
			return err
		}
		return nil
	},
}

var startCMD = cli.Command{
	Name:  "start",
	Usage: "Start fabric appchain plugin",
	Action: func(ctx *cli.Context) error {
		plugin.Serve(&plugin.ServeConfig{
			HandshakeConfig: plugins.Handshake,
			Plugins: map[string]plugin.Plugin{
				plugins.PluginName: &plugins.AppchainGRPCPlugin{Impl: &Client{}},
			},
			Logger:     logger,
			GRPCServer: plugin.DefaultGRPCServer,
		})

		logger.Info("Plugin server down")

		return nil
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "fabric-plugin"
	app.Usage = "Manipulate the fabric blockchain"
	app.Compiled = time.Now()

	app.Commands = []cli.Command{
		initCMD,
		startCMD,
	}

	err := app.Run(os.Args)
	if err != nil {
		color.Red(err.Error())
		os.Exit(-1)
	}
}
