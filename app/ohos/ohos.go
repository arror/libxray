package ohos

import (
	"fmt"

	common "github.com/xtls/xray-core/common"
)

var instance PlatformSupport

type PlatformSupport interface {
	Log(string) error
	GetDefaultNetInterfaceName() (string, error)
}

func RegisterPlatformSupport(ps PlatformSupport) {
	instance = ps
}

func GetPlatformSupport() (PlatformSupport, error) {
	if instance == nil {
		return nil, fmt.Errorf("PlatformSupport not register.")
	}
	return instance, nil
}

func MustGetPlatformSupport() PlatformSupport {
	return common.Must2(GetPlatformSupport()).(PlatformSupport)
}
