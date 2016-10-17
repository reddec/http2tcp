// Package h2t allows expose your fancy application over
// http CONNECT tunnel.
//
// This utility/library has very simple idea: it brings up http server and route CONNECT requests
// to backend over TCP connection
//
// Let me show you possible benefit:
//
// For example we want to connect to our SSH over internet, but we can only do it through
// http connection.
//
// Solution:
//
// ```
//
// +----------+                                    +----------+
// |          |         Http proxy                 |          |            +------+
// | Client   | -- CONNECT server:22 HTTP/1.1 -->  | http2tcp | -- TCP --> | SSHD |
// |          |                                    |          |            +------+
// +----------+                                    +----------+
//
// ```
//
//
// You have to ask me: why do this if we have Squid? Answer is simple: because it's much
// simple to configure and control access. Remote client can connect only to specified
// address
package h2t

import (
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

type rule struct {
	Target string
}

// Rules of CONNECT -> TCP backend
type Rules struct {
	rules map[string]rule
}

// Add single rule into table. Doesn't affect opened connections
func (rules *Rules) Add(servceName, targetAddress string) {
	if rules.rules == nil {
		rules.rules = make(map[string]rule)
	}
	rules.rules[servceName] = rule{targetAddress}
}

// Remove single rule from table. Doesn't affect opened connections
func (rules *Rules) Remove(servceName string) {
	delete(rules.rules, servceName)
}

// Clean all rules. Doesn't affect opened connections
func (rules *Rules) Clean() {
	rules.rules = make(map[string]rule)
}

// Table (copy) of rules: service -> target address
func (rules *Rules) Table() map[string]string {
	res := make(map[string]string)
	for k, v := range rules.rules {
		res[k] = v.Target
	}
	return res
}

// ServeHTTP - HTTP handler that accepts connection, check CONNECT method,
// find service and connect to tcp
func (rules *Rules) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "CONNECT" {
		http.Error(w, "Expect CONNECT method", http.StatusBadRequest)
		return
	}
	log.Println("New connection from ", r.RemoteAddr, "to service", r.RequestURI)
	rule, found := rules.rules[r.RequestURI]
	if !found {
		http.Error(w, "Service "+r.RequestURI+" not found", http.StatusNotFound)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Server does not supports hijack", http.StatusInternalServerError)
		return
	}

	addr, err := net.ResolveTCPAddr("tcp", rule.Target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer conn.Close()
	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(5 * time.Second)
	w.WriteHeader(200)
	srvconn, rw, err := hj.Hijack()
	if !ok {
		http.Error(w, "hijack failed", http.StatusInternalServerError)
		return
	}
	defer srvconn.Close()
	go func() {
		io.Copy(rw, conn)
		srvconn.Close()
	}()

	io.Copy(conn, rw)
	conn.Close()
	log.Println("Connection from ", r.RemoteAddr, "to service", r.RequestURI, "closed")
}

func NewRules() *Rules {
	return &Rules{make(map[string]rule)}
}
