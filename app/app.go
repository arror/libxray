package app

import (
	"bytes"
	"encoding/json"
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

type Config struct {
	FilesDir string `json:"filesDir"`
	CacheDir string `json:"cacheDir"`
	TempDir  string `json:"tempDir"`
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
	instance.AddFeature(common.Must1(core.CreateObject(instance, &server.Config{
		Path: filepath.Join(config.FilesDir, "vpn.sock"),
	})).(features.Feature))
	temp := struct {
		Tun tun.Config `json:"tun"`
	}{}
	err = json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	instance.AddFeature(common.Must1(core.CreateObject(instance, &temp.Tun)).(features.Feature))
	return instance.Start()
}
