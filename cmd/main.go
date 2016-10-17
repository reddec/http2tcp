package main

import (
	"bytes"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/reddec/http2tcp"
)

var bind = flag.String("b", ":9000", "Binding port")

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
	log.Fatal(http.ListenAndServe(*bind, rules))
}
