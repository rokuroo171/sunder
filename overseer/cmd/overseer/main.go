// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"

	"sunder/overseer/internal/console"
	"sunder/overseer/internal/whisper"
)

var version = "0.2.0"

func main() {
	listen := flag.String("listen", ":8443", "HTTPS listen address")
	headless := flag.Bool("headless", false, "serve without the console")
	forceConsole := flag.Bool("console", false, "run the console even when stdin is not a tty")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("sunder-overseer %s\n", version)
		return
	}

	cert, err := selfSignedCert()
	if err != nil {
		log.Fatalf("overseer: %v", err)
	}
	server := whisper.NewServer()
	events := make(chan string, 128)
	server.SetEventHandler(func(msg string) {
		select {
		case events <- msg:
		default:
		}
	})
	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for range t.C {
			server.Reap()
		}
	}()
	hs := &http.Server{
		Handler:   server,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12},
	}
	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("overseer: %v", err)
	}
	fmt.Printf("* Overseer (the hand) listening on https://%s\n", ln.Addr())
	fmt.Printf("* verse: a shard will come knocking\n")

	serveErr := make(chan error, 1)
	go func() { serveErr <- hs.ServeTLS(ln, "", "") }()

	if !*headless && (*forceConsole || isTTY(os.Stdin)) {
		c := console.New(server, os.Stdin, os.Stdout, events)
		if err := c.Run(); err != nil {
			log.Printf("console: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = hs.Shutdown(ctx)
		return
	}
	if err := <-serveErr; err != nil {
		log.Fatal(err)
	}
}

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// selfSignedCert builds an ephemeral certificate so a dev handshake works out of the box
func selfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "sunder-overseer"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return tls.X509KeyPair(certPEM, keyPEM)
}
