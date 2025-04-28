package main

/*
#include <stdlib.h>
int OHOS_LOG(size_t level, const char *message);
int OHOS_GetDefaultNetInterfaceName(char* name, size_t size);
*/
import "C"
import (
	"bytes"
	"fmt"
	"unsafe"

	"vpn/app"
	"vpn/app/ohos"
)

func main() {
}

//export Run
func Run(config []byte) int32 {
	if err := app.Run(config); err != nil {
		ohos.MustGetPlatformSupport().Log(fmt.Sprintf("Run Error: %v", err))
		return 500
	}
	return 0
}

type OHOSSupport struct{}

func (s *OHOSSupport) Log(message string) error {
	msg := C.CString(message)
	defer C.free(unsafe.Pointer(msg))
	C.OHOS_LOG(3, msg)
	return nil
}

func (s *OHOSSupport) GetDefaultNetInterfaceName() (string, error) {
	name := make([]byte, 1024)
	ret := C.OHOS_GetDefaultNetInterfaceName(
		(*C.char)(unsafe.Pointer(&name[0])),
		C.size_t(len(name)),
	)
	if ret != 0 {
		return "", fmt.Errorf("获取默认网络接口名称失败: %d", ret)
	}
	return string(bytes.TrimRight(name, "\x00")), nil
}

func init() {
	ohos.RegisterPlatformSupport(&OHOSSupport{})
}
