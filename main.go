package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
)

var (
	// 代理监听主机
	ProxyHost = "127.0.0.1"
	// 代理监听端口
	ProxyPort = 8080
	// 需要反代的网址
	SiteList = []string{"pixiv.net", "*.pixiv.net", "*.secure.pixiv.net", "*.pximg.net", "*.pixiv.org",
		"github.com", "*.github.com", "*.githubusercontent.com", "*.githubassets.com"}
	// DOH 列表
	DohList = []string{
		"https://dns.artikel10.org/dns-query",
		"https://dns1.dnscrypt.ca:453/dns-query",
		"https://dns.digitalsize.net/dns-query",
	}

	// 正代与反代间通信不需要占用端口
	FakeListener = &Listener{make(chan net.Conn, 100)}
	// DNS 缓存
	DnsCache = &sync.Map{}
)

func init() {
	gencert("./", SiteList)
	genpac("./", PacParam{SiteList, ProxyHost, ProxyPort})
}

// A Listener is a generic network listener for stream-oriented protocols.
// Multiple goroutines may invoke methods on a Listener simultaneously.
type Listener struct {
	channel chan net.Conn
}

// Accept waits for and returns the next connection to the listener.
func (ln *Listener) Accept() (net.Conn, error) {
	conn := <-ln.channel
	return conn, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (ln *Listener) Close() error {
	close(ln.channel)
	return nil
}

// Addr returns the listener's network address.
func (ln *Listener) Addr() net.Addr {
	return &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)}
}

func main() {
	// 反向代理
	log.Println("PixivChan OK!")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				return lookup(DohList, DnsCache, r.Host)
			},
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Println(r.Host, r.RequestURI, err)
		}
		proxy.ServeHTTP(w, r)
	})
	go func() {
		srv := &http.Server{Addr: "Go!", Handler: handler}
		err := srv.ServeTLS(FakeListener, "pixivchan.cer", "pixivchan.key")
		log.Panicln(err)
	}()

	// 正向代理，使流量走到反向代理
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ProxyHost, ProxyPort))
	if err != nil {
		log.Panicln(err)
	}
	for {
		browser, err := ln.Accept()
		if err != nil {
			log.Panicln(err)
		}
		go func(browser net.Conn) {
			defer browser.Close()
			var b = make([]byte, 1024)
			n, err := browser.Read(b)
			if err != nil {
				log.Println(err)
				return
			}
			var method, host, address string
			fmt.Sscanf(string(b[:n]), "%s%s", &method, &host)
			uri, err := url.Parse(host)
			if err != nil {
				log.Println(err)
				return
			}
			if method == "CONNECT" { // HTTPS
				address = uri.Scheme + ":" + uri.Opaque
			} else { // HTTP
				address = uri.Host
				if !strings.Contains(uri.Host, ":") {
					address = uri.Host + ":80"
				}
			}
			var server net.Conn
			if inpac(SiteList, uri.Scheme) {
				var reverse net.Conn
				server, reverse = net.Pipe()
				FakeListener.channel <- reverse
			} else {
				server, err = net.Dial("tcp", address)
				if err != nil {
					log.Println(err)
					return
				}
			}
			if method == "CONNECT" {
				fmt.Fprint(browser, "HTTP/1.1 200 Connection established\r\n\r\n")
			} else {
				server.Write(b[:n])
			}
			go io.Copy(server, browser)
			io.Copy(browser, server)
		}(browser)
	}
}
