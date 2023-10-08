package request

import (
	"fmt"
	"net"
	"net/http"
)

type requestInfo struct {
	RemoteAddr    string
	IP            string
	Port          string
	XForwardedFor string `json:"X-Forwarded-For"`

	RequestedHost string
	RequestedPort string
}

func (ri *requestInfo) String() string {
	return fmt.Sprintf("<RemoteAddr: %s; IP: %s; Port: %s; X-Forwarded-For: %s>", ri.RemoteAddr, ri.IP, ri.Port, ri.XForwardedFor)
}

func GetRequestInfo(r *http.Request) *requestInfo {
	ri := &requestInfo{RemoteAddr: r.RemoteAddr}
	ri.IP, ri.Port, _ = net.SplitHostPort(r.RemoteAddr)
	// assumes no proxy (r.URL.Host)
	ri.RequestedHost, ri.RequestedHost, _ = net.SplitHostPort(r.Host)
	return ri
}
