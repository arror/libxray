package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"vpn/app/server"
	"vpn/app/tun"

	"github.com/sagernet/sing/common"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features"

	"github.com/xtls/xray-core/common/platform"
)

var instance *core.Instance

type VpnConfig struct {
	MTU int `json:"mtu"`
}

type SniffingObject struct {
	Enabled                        bool     `json:"enbale"`
	OverrideDestinationForProtocol []string `json:"overrideDestinationForProtocol"`
	MetadataOnly                   bool     `json:"metadataOnly"`
	RouteOnly                      bool     `json:"routeOnly"`
}

type InboundObject struct {
	Tag      string         `json:"tag"`
	Fd       int            `json:"fd"`
	Config   VpnConfig      `json:"config"`
	Sniffing SniffingObject `json:"sniffing"`
}

type Config struct {
	Id       string        `json:"id"`
	Inbound  InboundObject `json:"inbound"`
	FilesDir string        `json:"filesDir"`
	CacheDir string        `json:"cacheDir"`
	TempDir  string        `json:"tempDir"`
}

func Run(config []byte) (err error) {
	cfg := Config{}
	err = json.Unmarshal(config, &cfg)
	if err != nil {
		return err
	}
	os.Setenv(platform.AssetLocation, fmt.Sprintf("%s/asset", cfg.FilesDir))
	return run(cfg)
}

func run(config Config) (err error) {
	data, err := os.ReadFile(fmt.Sprintf("%s/%s.json", config.TempDir, config.Id))
	if err != nil {
		return err
	}
	cfg, err := core.LoadConfig("json", bytes.NewReader(data))
	cfg.Inbound = []*core.InboundHandlerConfig{}
	if err != nil {
		return err
	}
	instance, err = core.New(cfg)
	if err != nil {
		return err
	}
	instance.AddFeature(common.Must1(core.CreateObject(instance, &server.Config{
		Path: filepath.Join(config.FilesDir, "vpn.sock"),
	})).(features.Feature))
	instance.AddFeature(common.Must1(core.CreateObject(instance, &tun.Config{
		Tag: config.Inbound.Tag,
		Fd:  config.Inbound.Fd,
		MTU: config.Inbound.Config.MTU,
		Sniffing: tun.SniffingObject{
			Enabled:                        config.Inbound.Sniffing.Enabled,
			OverrideDestinationForProtocol: config.Inbound.Sniffing.OverrideDestinationForProtocol,
			MetadataOnly:                   config.Inbound.Sniffing.MetadataOnly,
			RouteOnly:                      config.Inbound.Sniffing.RouteOnly,
		},
	})).(features.Feature))
	return instance.Start()
}
