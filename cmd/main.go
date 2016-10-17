package main

import (
	"bytes"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"sync"

	"github.com/gin-gonic/gin"
	"github.com/reddec/http2tcp"
)

func loadRules(content string) *h2t.Rules {
	r := h2t.NewRules()
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
		r.Add(service, target)
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

type newRule struct {
	Service string `json:"service" form:"service" xml:"service" yaml:"service"`
	Target  string `json:"target" form:"target" xml:"target" yaml:"target"`
}

type apiHandler struct {
	sync.RWMutex
	rules *h2t.Rules
}

func (api *apiHandler) items(c *gin.Context) {
	api.RLock()
	defer api.RUnlock()
	c.JSON(200, api.rules.Table())
}

func (api *apiHandler) add(c *gin.Context) {
	api.Lock()
	defer api.Unlock()

	req := newRule{}
	if err := c.Bind(&req); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	api.rules.Add(req.Service, req.Target)
	c.AbortWithStatus(http.StatusNoContent)
}

func (api *apiHandler) remove(c *gin.Context) {
	api.Lock()
	defer api.Unlock()
	api.rules.Remove(c.Param("name"))
	c.AbortWithStatus(http.StatusNoContent)
}

func (api *apiHandler) clean(c *gin.Context) {
	api.Lock()
	defer api.Unlock()
	api.rules.Clean()
	c.AbortWithStatus(http.StatusNoContent)
}

var bind = flag.String("b", ":9000", "Binding address")
var noApi = flag.Bool("no-api", false, "Disable access router configuration over HTTP")

type router struct {
	Proxy http.Handler
	Other http.Handler
}

func (rt *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "CONNECT" {
		rt.Proxy.ServeHTTP(w, r)
	} else {
		rt.Other.ServeHTTP(w, r)
	}
}

func main() {
	flag.Parse()
	var rules *h2t.Rules = h2t.NewRules()
	if len(flag.Args()) == 0 {
		if *noApi {
			log.Fatal("At least one configuration file must be provided as positional argument or API enabled")
		}
	} else {
		templ, err := template.ParseFiles(flag.Args()...)
		if err != nil {
			log.Fatal("Failed preprocess configuration files", err)
		}
		buffer := &bytes.Buffer{}
		err = templ.Execute(buffer, getEnv())
		if err != nil {
			log.Fatal("Failed execute proceprocessor", err)
		}
		rules = loadRules(buffer.String())
	}
	r := gin.Default()
	if !*noApi {
		handler := &apiHandler{rules: rules}
		r.GET("/api/", handler.items)
		r.DELETE("/api/", handler.clean)
		r.POST("/api/", handler.add)
		r.DELETE("/api/:name/", handler.remove)
	}
	log.Fatal(http.ListenAndServe(*bind, &router{Proxy: rules, Other: r}))
}
