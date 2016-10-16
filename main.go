package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
)

type Rule struct {
	Target string
}

type Rules struct {
	rules map[string]Rule
}

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

var bind = flag.String("b", ":9000", "Binding port")

func loadRules(content string) Rules {
	r := Rules{make(map[string]Rule)}
	lines := strings.Split(content, "\n")
	for n, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if line[0] == '#' {
			continue
		}
		if strings.Index(line, " ") == -1 {
			log.Fatal("Bad line #", n)
		}
		st := strings.SplitN(line, " ", 2)
		service, target := strings.TrimSpace(st[0]), strings.TrimSpace(st[1])
		r.rules[service] = Rule{target}
		log.Println("Add service", service, "pointed to", target)
	}
	return r
}

func getEnv() map[string]string {
	res := map[string]string{}
	for _, env := range os.Environ() {
		kv := strings.SplitN(env, "=", 2)
		res[kv[0]] = kv[1]
	}
	return res
}

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatal("At least one configuration file must be provided as positional argument")
	}

	templ, err := template.ParseFiles(flag.Args()...)
	if err != nil {
		log.Fatal("Failed preprocess configuration files", err)
	}
	buffer := &bytes.Buffer{}
	err = templ.Execute(buffer, getEnv())
	if err != nil {
		log.Fatal("Failed execute proceprocessor", err)
	}
	rules := loadRules(buffer.String())
	log.Fatal(http.ListenAndServe(*bind, &rules))
}
