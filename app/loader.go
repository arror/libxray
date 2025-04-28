package app

import (
	"io"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"
)

func init() {
	common.Must(core.RegisterConfigLoader(&core.ConfigFormat{
		Name:      "JSON",
		Extension: []string{"json"},
		Loader: func(input any) (*core.Config, error) {
			switch v := input.(type) {
			case io.Reader:
				return serial.LoadJSONConfig(v)
			default:
				panic("unavailable")
			}
		},
	}))
}
