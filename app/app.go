package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"vpn/app/server"
	"vpn/app/tun"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features"

	"github.com/xtls/xray-core/common/platform"
	"github.com/xtls/xray-core/common/session"
)

var instance *core.Instance

type Config struct {
	FilesDir string `json:"filesDir"`
	CacheDir string `json:"cacheDir"`
	TempDir  string `json:"tempDir"`
}

type TunSniffingConfig struct {
	ExcludeForDomain               []string `json:"domainsExcluded"`
	OverrideDestinationForProtocol []string `json:"destOverride"`
	Enabled                        bool     `json:"enabled"`
	MetadataOnly                   bool     `json:"metadataOnly"`
	RouteOnly                      bool     `json:"routeOnly"`
}

type TunConfig struct {
	Tag      string            `json:"tag"`
	Fd       int               `json:"fd"`
	MTU      int               `json:"mtu"`
	Sniffing TunSniffingConfig `json:"sniffing"`
}

func Run(config []byte) (err error) {
	cfg := Config{}
	err = json.Unmarshal(config, &cfg)
	if err != nil {
		return err
	}
	os.Setenv(platform.AssetLocation, filepath.Join(cfg.FilesDir, "asset"))
	return run(cfg)
}

func run(config Config) (err error) {
	data, err := os.ReadFile(filepath.Join(config.TempDir, "config.json"))
	if err != nil {
		return err
	}
	cfg, err := core.LoadConfig("json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	instance, err = core.New(cfg)
	if err != nil {
		return err
	}
	instance.AddFeature(common.Must2(core.CreateObject(instance, &server.Config{
		Path: filepath.Join(config.FilesDir, "vpn.sock"),
	})).(features.Feature))
	temp := &struct {
		Tun TunConfig `json:"tun"`
	}{}
	err = json.Unmarshal(data, temp)
	if err != nil {
		return err
	}
	instance.AddFeature(common.Must2(core.CreateObject(instance, &tun.Config{
		Tag: temp.Tun.Tag,
		Fd:  temp.Tun.Fd,
		MTU: temp.Tun.MTU,
		Sniffing: session.SniffingRequest{
			Enabled:                        temp.Tun.Sniffing.Enabled,
			MetadataOnly:                   temp.Tun.Sniffing.MetadataOnly,
			RouteOnly:                      temp.Tun.Sniffing.RouteOnly,
			OverrideDestinationForProtocol: temp.Tun.Sniffing.OverrideDestinationForProtocol,
			ExcludeForDomain:               temp.Tun.Sniffing.ExcludeForDomain,
		},
	})).(features.Feature))
	return instance.Start()
}
