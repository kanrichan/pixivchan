package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

// 需要反代的站点及 DNS Name
var SiteList = map[string][]string{
	"pixiv.net": {"pixiv.net", "*.pixiv.net", "*.secure.pixiv.net"},
	"pximg.net": {"pximg.net", "*.pximg.net"},
	"pixiv.org": {"pixiv.org", "*.pixiv.org"},
	"github.com": {"github.com", "*.github.com", "githubusercontent.com",
		"*.githubusercontent.com", "githubassets.com", "*.githubassets.com"},
}

// 证书列表
var Certs = make([]*tls.Certificate, 0)

// 找不到对应的证书
var CertNotFoundErr = errors.New("x509 certificate not found")

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

// 加载证书
func init() {
	var dir = "./"
	// 加载或生成 CA 证书
	if _, err := os.Stat(path.Join(dir, "ca.cer")); os.IsNotExist(err) {
		if err := signCA(dir); err != nil {
			panic(err)
		}
	}
	_, b1, err := loadpem(path.Join(dir, "ca.cer"))
	if err != nil {
		panic(err)
	}
	cacert, err := x509.ParseCertificate(b1)
	if err != nil {
		panic(err)
	}
	_, b2, err := loadpem(path.Join(dir, "ca.key"))
	if err != nil {
		panic(err)
	}
	cakey, err := x509.ParseECPrivateKey(b2)
	if err != nil {
		panic(err)
	}
	// 加载或生成反代站点证书
	for site, dns := range SiteList {
		if _, err := os.Stat(path.Join(dir, site+".cer")); os.IsNotExist(err) {
			err := signCert(dir, site, dns, cacert, cakey)
			if err != nil {
				panic(err)
			}
		}
		cert, err := tls.LoadX509KeyPair(path.Join(dir, site+".cer"),
			path.Join(dir, site+".key"))
		if err != nil {
			panic(err)
		}
		leaf, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			panic(err)
		}
		cert.Leaf = leaf
		Certs = append(Certs, &cert)
		fmt.Println("Load", cert.Leaf.DNSNames)
	}
	DnsCache.Store("accounts.pixiv.net", "210.140.92.187")
}

func main() {
	http.HandleFunc("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url, err := url.Parse(fmt.Sprintf("https://%s", r.Host))
		if err != nil {
			panic(err)
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
				DnsCache.Range(func(key, value any) bool {
					return true
				})
				// 读取 DNS 缓存
				var host = strings.TrimPrefix(r.Host, "www.")
				if ip, ok := DnsCache.Load(host); ok {
					return net.Dial("tcp", ip.(string)+":443")
				}
				// 通过 DOH 获取正确的 IP
				for i := range DohList {
					data, err := doh(DohList[i], host)
					if err != nil {
						continue
					}
					for _, answer := range data.Answer {
						if answer.Data == "" || answer.Data == "127.0.0.1" ||
							len(answer.Data) < 7 || len(answer.Data) > 15 {
							continue
						}
						DnsCache.Store(host, answer.Data)
						DnsCache.Range(func(key, value any) bool {
							fmt.Println("DOH", key, value)
							return true
						})
						return net.Dial("tcp", answer.Data+":443")
					}
				}
				return nil, DohNotFoundErr
			},
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			fmt.Println("ERROR:", r.Host, r.RequestURI, err)
		}
		proxy.ServeHTTP(w, r)
	}))

	server := &http.Server{
		Addr: "127.0.0.1:443",
		TLSConfig: &tls.Config{
			GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
				for i := range Certs {
					for _, domain := range Certs[i].Leaf.DNSNames {
						if domain == chi.ServerName {
							return Certs[i], nil
						}
						ds := strings.Split(domain, ".")
						ss := strings.Split(chi.ServerName, ".")
						if len(ds) != len(ss) {
							continue
						}
						for j := len(ds) - 1; j >= 0; j-- {
							if ds[j] != ss[j] && ds[j] != "*" {
								break
							}
							if j == 0 {
								return Certs[i], nil
							}
						}
					}
				}
				return nil, CertNotFoundErr
			},
		},
	}

	err := server.ListenAndServeTLS("", "")
	panic(err)
}

func savepem(name, typ string, data []byte) error {
	block := pem.Block{
		Type:    typ,
		Headers: nil,
		Bytes:   data,
	}
	fi, err := os.Create(name)
	if err != nil {
		return err
	}
	defer fi.Close()
	return pem.Encode(fi, &block)
}

func loadpem(name string) (string, []byte, error) {
	fi, err := os.Open(name)
	if err != nil {
		return "", nil, err
	}
	b, err := ioutil.ReadAll(fi)
	if err != nil {
		return "", nil, err
	}
	p, _ := pem.Decode(b)
	return p.Type, p.Bytes, nil
}

func signCA(dir string) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Issuer:       pkix.Name{},
		Subject:      pkix.Name{Organization: []string{"FloatTech"}, CommonName: "PixivChan CA"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour * 24 * 365),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		IsCA:         true,
	}
	certificate, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}
	if err := savepem(path.Join(dir, "ca.cer"), "CERTIFICATE", certificate); err != nil {
		return err
	}
	ecpriv, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return err
	}
	if err := savepem(path.Join(dir, "ca.key"), "ECDSA PRIVATE KEY", ecpriv); err != nil {
		return err
	}
	return nil
}

func signCert(dir string, site string, dns []string, cacert *x509.Certificate, cakey *ecdsa.PrivateKey) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"FloatTech"}, CommonName: site},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour * 24 * 365),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		DNSNames:     dns,
	}
	certificate, err := x509.CreateCertificate(rand.Reader, &template, cacert, &priv.PublicKey, cakey)
	if err != nil {
		return err
	}
	if err := savepem(path.Join(dir, site+".cer"), "CERTIFICATE", certificate); err != nil {
		return err
	}
	ecpriv, err := x509.MarshalECPrivateKey(priv)
	if err := savepem(path.Join(dir, site+".key"), "ECDSA PRIVATE KEY", ecpriv); err != nil {
		return err
	}
	return nil
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
