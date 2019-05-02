package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/anynines/go-proxy-setup-ntlm/proxysetup/ntlm"
)

var corpProxyAddr = flag.String("corp-proxy", "corp.proxy.net:3128", "The addr of the ntlm proxy.")
var hopProxyAddr = flag.String("hop-proxy", "https://user:pass@your.proxy:443", "The hop proxy that allows CONNECT to anything")
var destAddr = flag.String("dest", "dest-host:1234", "The addr endpoint to connect to")

func init() {
	flag.Parse()
}

func handleConn(localConn io.ReadWriteCloser) {
	// connect corp proxy
	dialer := &net.Dialer{
		KeepAlive: 30 * time.Second,
		Timeout:   30 * time.Second,
	}
	remoteConn, err := dialer.Dial("tcp", *corpProxyAddr)
	if err != nil {
		log.Fatalln("error dial:", err)
		return
	}
	hopProxyUrl, err := url.Parse(*hopProxyAddr)
	if err != nil {
		log.Fatalln("error parsing proxy addr:", err)
		return
	}
	err = ntlm.ProxySetup(remoteConn, hopProxyUrl.Host)
	if err != nil {
		log.Fatalln("error proxy injection:", err)
		return
	}
	if hopProxyUrl.Scheme == "https" {
		// create ssl to my proxy
		sslConn := tls.Client(remoteConn, &tls.Config{
			ServerName: hopProxyUrl.Hostname(),
		})
		remoteConn = sslConn
		err = sslConn.Handshake()
	}
	// CONNECT via my proxy to some test server
	connectLine := "CONNECT " + *destAddr + " HTTP/1.1\n"
	if hopProxyUrl.User != nil {
		base64credentials := base64.StdEncoding.EncodeToString([]byte(hopProxyUrl.User.String()))
		connectLine += "Proxy-Authorization: Basic " + base64credentials + "\n"
	}
	connectLine += "\n"
	_, err = remoteConn.Write([]byte(connectLine))
	if err != nil {
		log.Fatalln("error writing connect:", err)
		return
	}
	buffer := make([]byte, 256)
	n, err := remoteConn.Read(buffer)
	if err != nil {
		log.Fatalln("error reading connect:", err)
		return
	}
	if !bytes.HasPrefix(buffer, []byte("HTTP/1.1 200 OK")) {
		log.Println("no HTTP OK", n, string(buffer), err)
		return
	}
	for n == len(buffer) {
		n, _ = remoteConn.Read(buffer)
	}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		io.Copy(localConn, remoteConn)
		remoteConn.Close()
		localConn.Close()
		wg.Done()
	}()
	go func() {
		io.Copy(remoteConn, localConn)
		remoteConn.Close()
		localConn.Close()
		wg.Done()
	}()
	wg.Wait()
}

type inout struct {
	in  io.ReadCloser
	out io.WriteCloser
}

func (x inout) Close() error {
	err := x.in.Close()
	err2 := x.out.Close()
	if err != nil {
		return err
	}
	return err2
}

func (x inout) Read(b []byte) (n int, err error) {
	return x.in.Read(b)
}

func (x inout) Write(b []byte) (n int, err error) {
	return x.out.Write(b)
}

func main() {
	handleConn(inout{
		in:  os.Stdin,
		out: os.Stdout,
	})
}
