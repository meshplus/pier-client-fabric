package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr"
	"github.com/hashicorp/go-plugin"
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
