package tcp_conn_server

import (
	"context"
	"math"
	"net"
)

const abortIndex int8 = math.MaxInt8

type TcpHandlerFunc func(ctx *TcpRouterContext)

type TcpRouterGroup struct {
	*TcpRouter
	path     string
	handlers []TcpHandlerFunc
}

func (g *TcpRouterGroup) Use(middlewares ...TcpHandlerFunc) *TcpRouterGroup {
	g.handlers = append(g.handlers, middlewares...)
	isExisted := false
	for _, group := range g.TcpRouter.groups {
		if group == g {
			isExisted = true
		}
	}
	if !isExisted {
		g.TcpRouter.groups = append(g.TcpRouter.groups, g)
	}
	return g
}

type TcpRouter struct {
	groups []*TcpRouterGroup
}

func NewTcpRouter() *TcpRouter {
	return &TcpRouter{}
}

func (r *TcpRouter) Group(path string) *TcpRouterGroup {
	return &TcpRouterGroup{
		TcpRouter: r,
		path:      path,
	}
}

type TcpRouterContext struct {
	*TcpRouterGroup
	conn  net.Conn
	Ctx   context.Context
	index int8
}

func NewTcpRouterContext(conn net.Conn, r *TcpRouter, ctx context.Context) *TcpRouterContext {
	tcpGroup := &TcpRouterGroup{}
	*tcpGroup = *r.groups[0]
	tcpCtx := &TcpRouterContext{
		TcpRouterGroup: tcpGroup,
		conn:           conn,
		Ctx:            ctx,
		index:          0,
	}

	return tcpCtx
}

func (ctx *TcpRouterContext) Get(key interface{}) interface{} {
	return ctx.Ctx.Value(key)
}

func (ctx *TcpRouterContext) Set(key, val interface{}) {
	ctx.Ctx = context.WithValue(ctx.Ctx, key, val)
}

func (ctx *TcpRouterContext) Reset() {
	ctx.index = -1
}

func (ctx *TcpRouterContext) Next() {
	ctx.index++
	for ctx.index < int8(len(ctx.handlers)) {
		ctx.handlers[ctx.index](ctx)
		ctx.index++
	}
}

func (ctx *TcpRouterContext) Abort() {
	ctx.index = abortIndex
}

func (ctx *TcpRouterContext) IsAborted() bool {
	return ctx.index >= abortIndex
}

type TcpRouterHandler struct {
	router   *TcpRouter
	coreFunc func(ctx *TcpRouterContext) TCPHandler
}

func NewTcpRouteHandler(router *TcpRouter, coreFunc func(ctx *TcpRouterContext) TCPHandler) *TcpRouterHandler {
	return &TcpRouterHandler{
		router:   router,
		coreFunc: coreFunc,
	}
}

func (h *TcpRouterHandler) ServeTcp(ctx context.Context, conn net.Conn) {
	tcpCtx := NewTcpRouterContext(conn, h.router, ctx)
	tcpCtx.handlers = append(tcpCtx.handlers, func(c *TcpRouterContext) {
		h.coreFunc(c).ServeTCP(ctx, conn)
	})
	tcpCtx.Reset()
	tcpCtx.Next()
}
