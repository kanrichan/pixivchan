package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"time"
)

// 需要反代的站点及 DNS Name
var DnsList = []string{"pixiv.net", "*.pixiv.net", "*.secure.pixiv.net", "pximg.net", "*.pximg.net", "pixiv.org", "*.pixiv.org",
	"github.com", "*.github.com", "githubusercontent.com", "*.githubusercontent.com", "githubassets.com", "*.githubassets.com"}

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
	if _, err := os.Stat(path.Join(dir, "pixivchan.cer")); os.IsNotExist(err) {
		err := signCert(dir, "pixivchan", DnsList, cacert, cakey)
		if err != nil {
			panic(err)
		}
	}
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

func signCert(dir string, name string, dns []string, cacert *x509.Certificate, cakey *ecdsa.PrivateKey) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"FloatTech"}, CommonName: name},
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
	if err := savepem(path.Join(dir, name+".cer"), "CERTIFICATE", certificate); err != nil {
		return err
	}
	ecpriv, err := x509.MarshalECPrivateKey(priv)
	if err := savepem(path.Join(dir, name+".key"), "ECDSA PRIVATE KEY", ecpriv); err != nil {
		return err
	}
	return nil
}
