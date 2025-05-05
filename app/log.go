package app

import (
	"sync"

	"vpn/app/ohos"

	appLog "github.com/xtls/xray-core/app/log"
	common "github.com/xtls/xray-core/common"
	commonLog "github.com/xtls/xray-core/common/log"
)

var _ commonLog.Writer = (*HiLog)(nil)

type HiLog struct {
	mutex sync.Mutex
	write func(string) error
}

func CreateStdoutLogWriter(writer func(string) error) (commonLog.WriterCreator, error) {
	return func() commonLog.Writer {
		return &HiLog{write: writer}
	}, nil
}

func (log *HiLog) Write(s string) error {
	log.mutex.Lock()
	defer log.mutex.Unlock()
	return log.write(s)
}

func (log *HiLog) Close() error {
	log.mutex.Lock()
	defer log.mutex.Unlock()
	log.write = nil
	return nil
}

func init() {
	common.Must(appLog.RegisterHandlerCreator(appLog.LogType_Console, func(_ appLog.LogType, _ appLog.HandlerCreatorOptions) (commonLog.Handler, error) {
		return commonLog.NewLogger(
			common.Must2(CreateStdoutLogWriter(ohos.MustGetPlatformSupport().Log)).(commonLog.WriterCreator),
		), nil
	}))
}
