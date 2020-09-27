package gmcservice

import (
	"log"
	"net"

	gmcconfig "github.com/snail007/gmc/config/gmc"
)

type Service interface {
	Init(cfg *gmcconfig.GMCConfig) error
	Start() error
	Stop()
	GracefulStop()
	SetLog(*log.Logger)
	InjectListener(net.Listener)
	Listener() net.Listener
}