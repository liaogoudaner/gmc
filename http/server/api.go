package gmchttpserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	gmcconfig "github.com/snail007/gmc/config/gmc"
	gmchttputil "github.com/snail007/gmc/util/httputil"

	"github.com/snail007/gmc/util/logutil"

	gmcrouter "github.com/snail007/gmc/http/router"
)

type APIServer struct {
	listener          net.Listener
	server            *http.Server
	address           string
	router            *gmcrouter.HTTPRouter
	logger            *log.Logger
	handle404         func(ctx *gmcrouter.Ctx)
	handle500         func(ctx *gmcrouter.Ctx, err interface{})
	isShowErrorStack  bool
	certFile, keyFile string
	middleware0       []func(ctx *gmcrouter.Ctx, server *APIServer) (isStop bool)
	middleware1       []func(ctx *gmcrouter.Ctx, server *APIServer) (isStop bool)
	middleware2       []func(ctx *gmcrouter.Ctx, server *APIServer) (isStop bool)
	middleware3       []func(ctx *gmcrouter.Ctx, server *APIServer) (isStop bool)
	isShutdown        bool
}

func NewAPIServer(address string) *APIServer {
	api := &APIServer{
		server: &http.Server{
			TLSConfig: &tls.Config{},
		},
		address:          address,
		logger:           logutil.New(""),
		router:           gmcrouter.NewHTTPRouter(),
		isShowErrorStack: true,
		middleware0:      []func(ctx *gmcrouter.Ctx, server *APIServer) (isStop bool){},
		middleware1:      []func(ctx *gmcrouter.Ctx, server *APIServer) (isStop bool){},
		middleware2:      []func(ctx *gmcrouter.Ctx, server *APIServer) (isStop bool){},
		middleware3:      []func(ctx *gmcrouter.Ctx, server *APIServer) (isStop bool){},
	}
	api.server.Handler = api
	api.server.SetKeepAlivesEnabled(false)
	api.server.ErrorLog = api.logger
	return api
}

func (this *APIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w = gmchttputil.NewResponseWriter(w)
	c0 := gmcrouter.NewCtx(w, r)
	defer func() {
		// middleware3
		this.callMiddleware(c0, this.middleware3)
	}()
	//middleware0
	if this.callMiddleware(c0, this.middleware0) {
		return
	}

	h, args, _ := this.router.Lookup(r.Method, r.URL.Path)
	if h != nil {
		c := gmcrouter.NewCtx(w, r, args)
		c0 = c
		// middleware1
		if this.callMiddleware(c, this.middleware1) {
			return
		}

		status := ""
		err := this.call(func() { h(w, r, args) })
		if err != nil {
			status = fmt.Sprintf("%s", err)
		}
		switch status {
		case "__STOP__", "":
		default:
			//exception
			this.handler500(gmcrouter.NewCtx(w, r, args), err)
		}

		// middleware2
		if this.callMiddleware(c, this.middleware2) {
			return
		}

	} else {
		this.handler404(gmcrouter.NewCtx(w, r))
	}
}
func (this *APIServer) Server() *http.Server {
	return this.server
}
func (this *APIServer) SetTLSFile(certFile, keyFile string) *APIServer {
	this.certFile, this.keyFile = certFile, keyFile
	return this
}
func (this *APIServer) SetLogger(l *log.Logger) *APIServer {
	this.logger = l
	return this
}
func (this *APIServer) Logger() *log.Logger {
	return this.logger
}
func (this *APIServer) AddMiddleware0(m func(ctx *gmcrouter.Ctx, server *APIServer) (isStop bool)) *APIServer {
	this.middleware0 = append(this.middleware0, m)
	return this
}
func (this *APIServer) AddMiddleware1(m func(ctx *gmcrouter.Ctx, server *APIServer) (isStop bool)) *APIServer {
	this.middleware1 = append(this.middleware1, m)
	return this
}
func (this *APIServer) AddMiddleware2(m func(ctx *gmcrouter.Ctx, server *APIServer) (isStop bool)) *APIServer {
	this.middleware2 = append(this.middleware2, m)
	return this
}
func (this *APIServer) AddMiddleware3(m func(ctx *gmcrouter.Ctx, server *APIServer) (isStop bool)) *APIServer {
	this.middleware3 = append(this.middleware3, m)
	return this
}
func (this *APIServer) Handle404(handle func(ctx *gmcrouter.Ctx)) *APIServer {
	this.handle404 = handle
	return this
}
func (this *APIServer) Handle500(handle func(ctx *gmcrouter.Ctx, err interface{})) *APIServer {
	this.handle500 = handle
	return this
}
func (this *APIServer) ShowErrorStack(isShow bool) *APIServer {
	this.isShowErrorStack = isShow
	return this
}
func (this *APIServer) API(path string, handle func(ctx *gmcrouter.Ctx)) *APIServer {
	this.router.HandleAny(path, func(w http.ResponseWriter, r *http.Request, ps gmcrouter.Params) {
		handle(gmcrouter.NewCtx(w, r, ps))
	})
	return this
}
func (this *APIServer) Group(path string) *APIServer {
	newAPI := *this
	newAPI.router = this.router.Group(path)
	return &newAPI
}

// PrintRouteTable dump all routes into `w`, if `w` is nil, os.Stdout will be used.
func (this *APIServer) PrintRouteTable(w io.Writer) {
	this.router.PrintRouteTable(w)
}
func (this *APIServer) Run() (err error) {
	if this.listener == nil {
		this.listener, err = net.Listen("tcp", this.address)
		if err != nil {
			return
		}
	}
	this.address = this.listener.Addr().String()
	go func() {
		var err error
		if this.certFile != "" && this.keyFile != "" {
			this.logger.Printf("api server on https://%s", this.address)
			err = this.server.ServeTLS(this.listener, this.certFile, this.keyFile)
		} else {
			this.logger.Printf("api server on http://%s", this.address)
			err = this.server.Serve(this.listener)
		}
		if err != nil {
			if strings.Contains(err.Error(), "closed") {
				if this.isShutdown {
					this.logger.Printf("api server graceful shutdown on %s", this.address)
				} else {
					this.logger.Printf("api server closed on %s", this.address)
					this.server.Close()
				}
			} else {
				this.logger.Printf("api server exited unexpectedly on %s, error : %s", this.address, err)
			}
		}
	}()
	return
}
func (this *APIServer) handler404(ctx *gmcrouter.Ctx) *APIServer {
	if this.handle404 == nil {
		ctx.Response.WriteHeader(http.StatusNotFound)
		ctx.Response.Write([]byte("Page not found"))
	} else {
		this.handle404(ctx)
	}
	return this
}
func (this *APIServer) handler500(ctx *gmcrouter.Ctx, err interface{}) *APIServer {
	if this.handle500 == nil {
		ctx.WriteHeader(http.StatusInternalServerError)
		ctx.Response.Header().Set("Content-Type", "text/plain")
		msg := fmt.Sprintf("Internal Server Error")
		if err != nil && this.isShowErrorStack {
			msg += fmt.Sprintf("\n%s\n", err) + string(debug.Stack())
		}
		ctx.Write([]byte(msg))
	} else {
		this.handle500(ctx, err)
	}
	return this
}
func (s *APIServer) call(fn func()) (err interface{}) {
	func() {
		defer func() {
			err = recover()
		}()
		fn()
	}()
	return
}
func (s *APIServer) callMiddleware(ctx *gmcrouter.Ctx, middleware []func(ctx *gmcrouter.Ctx, server *APIServer) (isStop bool)) (isStop bool) {
	for _, fn := range middleware {
		func() {
			defer func() {
				if e := recover(); e != nil {
					s.logger.Printf("middleware pani error : %s", e)
					isStop = false
				}
			}()
			isStop = fn(ctx, s)
		}()
		if isStop {
			return
		}
	}
	return
}

//Init implements service.Services Init
func (s *APIServer) Init(cfg *gmcconfig.GMCConfig) (err error) {
	return
}

//Start implements service.Services Start
func (this *APIServer) Start() (err error) {
	this.Run()
	return
}

//Stop implements service.Services Stop
func (this *APIServer) Stop() {
	this.server.Close()
}

//GracefulStop implements service.Services GracefulStop
func (this *APIServer) GracefulStop() {
	if this.isShutdown {
		return
	}
	this.isShutdown = true
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	this.server.Shutdown(ctx)
	return
}

//SetLog implements service.Services SetLog
func (this *APIServer) SetLog(l *log.Logger) {
	this.logger = l
}

//InjectListener implements service.Services InjectListener
func (this *APIServer) InjectListener(l net.Listener) {
	this.listener = l
}

//Listener implements service.Services Listener
func (this *APIServer) Listener() net.Listener {
	return this.listener
}