package server

import (
	"github.com/kercylan98/minotaur/utils/log"
	"time"
)

type (
	RunMode = log.RunMode
)

const (
	RunModeDev  RunMode = log.RunModeDev
	RunModeProd RunMode = log.RunModeProd
	RunModeTest RunMode = log.RunModeTest
)

const (
	serverMultipleMark = "Minotaur Multiple Server"
	serverMark         = "Minotaur Server"
)

const (
	DefaultMessageBufferSize     = 1024
	DefaultMessageChannelSize    = 1024 * 64
	DefaultAsyncPoolSize         = 256
	DefaultWebsocketReadDeadline = 30 * time.Second
)
