package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func init() {
	DnsCache.Store("accounts.pixiv.net", "210.140.92.187")
}

func main() {
	log.Println("Reverse Proxy OK!")
	handle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Host, r.RequestURI, "start")
		url, err := url.Parse(fmt.Sprintf("https://%s", r.Host))
		if err != nil {
			log.Println(r.Host, r.RequestURI, err)
		}
		proxy := httputil.NewSingleHostReverseProxy(url)
		proxy.Transport = &http.Transport{
			DisableKeepAlives: true,
			// 隐藏 sni 标志
			TLSClientConfig: &tls.Config{
				ServerName:         "-",
				InsecureSkipVerify: true,
			},
			// 指向正确的 IP
			Dial: func(network, addr string) (net.Conn, error) {
				return lookup(DohList, DnsCache, r.Host) // 指向正确的 IP
			},
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Println(r.Host, r.RequestURI, err)
		}
		proxy.ServeHTTP(w, r)
	})
	err := http.ListenAndServeTLS("127.0.0.1:443", "pixivchan.cer", "pixivchan.key", handle)
	panic(err)
}
