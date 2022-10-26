package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DNS 缓存
var DnsCache = sync.Map{}

// Doh 列表
var DohList = []string{
	"https://dns.artikel10.org/dns-query",
	"https://dns.digitalsize.net/dns-query",
	"https://dns1.dnscrypt.ca:453/dns-query",
}

// Doh 找不到对应的主机记录
var DohNotFoundErr = errors.New("Doh cannot find matching record")

// Doh JSON 报文
type DohData struct {
	Status   int  `json:"Status"`
	Tc       bool `json:"TC"`
	Rd       bool `json:"RD"`
	Ra       bool `json:"RA"`
	Ad       bool `json:"AD"`
	Cd       bool `json:"CD"`
	Question []struct {
		Name string `json:"name"`
		Type int    `json:"type"`
	} `json:"Question"`
	Answer []struct {
		Name    string `json:"name"`
		Type    int    `json:"type"`
		TTL     int    `json:"TTL"`
		Expires string `json:"Expires"`
		Data    string `json:"data"`
	} `json:"Answer"`
	EdnsClientSubnet string `json:"edns_client_subnet"`
}

func doh(query string, host string) (*DohData, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?name=%s&type=A", query, host), nil)
	req.Header.Set("Accept", "application/dns-json")
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{Timeout: time.Second * 30}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data DohData
	var dec = json.NewDecoder(resp.Body)
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func lookup(dohs []string, cache sync.Map, host string) (net.Conn, error) {
	// 读取 DNS 缓存
	host = strings.TrimPrefix(host, "www.")
	if ip, ok := cache.Load(host); ok {
		conn, err := net.Dial("tcp", ip.(string)+":443")
		if err == nil {
			return conn, nil
		}
	}
	// 通过 DOH 获取正确的 IP
	for i := range dohs {
		data, err := doh(dohs[i], host)
		if err != nil {
			continue
		}
		for _, answer := range data.Answer {
			if answer.Data == "" || answer.Data == "127.0.0.1" ||
				len(answer.Data) < 7 || len(answer.Data) > 15 {
				continue
			}
			conn, err := net.Dial("tcp", answer.Data+":443")
			if err != nil {
				continue
			}
			DnsCache.Store(host, answer.Data)
			DnsCache.Range(func(key, value any) bool {
				return true
			})
			return conn, nil
		}
	}
	return nil, DohNotFoundErr
}
