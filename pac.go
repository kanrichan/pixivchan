package main

import (
	"html/template"
	"io"
	"strings"
)

// PAC 参数
type PacParam struct {
	SiteList  []string
	ProxyHost string
	ProxyPort int
}

// PAC 文件模板
var PacTemplate = `function FindProxyForURL(url, host) {
  {{range .SiteList}}
  if (shExpMatch(url,"*{{.}}/*")) {
    return "PROXY {{$.ProxyHost}}:{{$.ProxyPort}}";
  }
  {{- end}}
  return "DIRECT"; 
}`

func pac(wr io.Writer, param PacParam) error {
	pac, err := template.New("").Parse(PacTemplate)
	if err != nil {
		return err
	}
	err = pac.Execute(wr, param)
	if err != nil {
		return err
	}
	return nil
}

func inpac(list []string, host string) bool {
	var n1 = strings.Split(host, ".")
OUT:
	for i := range list {
		var n2 = strings.Split(list[i], ".")
		if len(n1) != len(n2) {
			continue
		}
		for j := range n1 {
			if n1[j] != n2[j] && n2[j] != "*" {
				continue OUT
			}
		}
		return true
	}
	return false
}
