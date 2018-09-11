package proxy

import (
	"encoding/binary"
	"fmt"
	"net"
	"syscall"

	"github.com/shell909090/goproxy/netutil"
)

const SO_ORIGINAL_DST = 80

type TransparentProxy struct {
	dialer     netutil.Dialer
	listenaddr string
}

func NewTransparentProxy(dialer netutil.Dialer, addr string) (tproxy *TransparentProxy) {
	tproxy = &TransparentProxy{
		dialer:     dialer,
		listenaddr: addr,
	}
	return
}

func (tproxy *TransparentProxy) Start() {
	logger.Infof("transparent start in %s", tproxy.listenaddr)
	go netutil.ListenAndServe("tcp", tproxy.listenaddr, tproxy.ServeConn)
}

func (tproxy *TransparentProxy) ServeConn(conn net.Conn) {
	var err error
	defer conn.Close()

	var rawconn syscall.RawConn

	switch tconn := conn.(type) {
	case *net.TCPConn:
		rawconn, err = tconn.SyscallConn()
	case *net.UDPConn:
		rawconn, err = tconn.SyscallConn()
	}
	if err != nil {
		logger.Error(err.Error())
		return
	}

	var str_addr string
	err = rawconn.Control(func(fd uintptr) {
		var addr *syscall.IPv6Mreq
		// int(fd), not feels so good...
		addr, err = syscall.GetsockoptIPv6Mreq(
			int(fd), syscall.IPPROTO_IP, SO_ORIGINAL_DST)
		if err != nil {
			logger.Error(err.Error())
			return
		}

		ipv4 := net.IP(addr.Multiaddr[4:8])
		port := binary.BigEndian.Uint16(addr.Multiaddr[2:4])
		str_addr = fmt.Sprintf("%s:%d", ipv4.String(), port)
	})
	if err != nil {
		logger.Error(err.Error())
		return
	}
	logger.Debugf("transparent connect to %s", str_addr)

	dstconn, err := tproxy.dialer.Dial("tcp", str_addr)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	// dstconn will be closed in CopyLink.

	netutil.CopyLink(conn, dstconn)
	return
}
