package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"io"
	/*
	"regexp"
	"strings"
	"github.com/mgutz/ansi"
	*/
)


var (
	connid = uint64(0)
	localHost = ":9876"
	remoteHost = flag.String("r", "localhost:5432", "Endereço do servidor PostgreSQL")
)


func main() {
	flag.Parse()
	fmt.Printf("Redirecionando de %v para %v\n", localHost, remoteHost)

	localAddr, remoteAddr := getEnderecosResolvidos(&localHost, remoteHost)
	listener := getListener(localAddr)

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			fmt.Printf("Erro ao aceitar conexão '%s'\n", err)
			continue
		}
		connid++

		p := &proxy{
			lconn:    conn,
			laddr:    localAddr,
			raddr:    remoteAddr,
			erred:    false,
			errsig:   make(chan bool),
			prefix:   fmt.Sprintf("Connection #%03d ", connid),
		}
		go p.start()
	}
}


func getEnderecosResolvidos(localHost, remoteHost *string) (*net.TCPAddr, *net.TCPAddr) {
	laddr, err := net.ResolveTCPAddr("tcp", *localHost)
	check(err)
	raddr, err := net.ResolveTCPAddr("tcp", *remoteHost)
	check(err)
	return laddr, raddr
}


func getListener(addr *net.TCPAddr) (*net.TCPListener) {
	listener, err := net.ListenTCP("tcp", addr)
	check(err)
	return listener
}


type proxy struct {
	sentBytes     uint64
	receivedBytes uint64
	laddr, raddr  *net.TCPAddr
	lconn, rconn  *net.TCPConn
	erred         bool
	errsig        chan bool
	prefix        string
}


func (p *proxy) err(s string, err error) {
	if p.erred {
		return
	}
	if err != io.EOF {
		warn(p.prefix+s, err)
	}
	p.errsig <- true
	p.erred = true
}


func (p *proxy) start() {
	defer p.lconn.Close()
	//connect to remote
	rconn, err := net.DialTCP("tcp", nil, p.raddr)
	if err != nil {
		p.err("Remote connection failed: %s", err)
		return
	}
	p.rconn = rconn
	defer p.rconn.Close()
	//bidirectional copy
	go p.pipe(p.lconn, p.rconn)
	go p.pipe(p.rconn, p.lconn)
	//wait for close...
	<-p.errsig
}


func (p *proxy) pipe(src, dst *net.TCPConn) {
	//data direction
	islocal := src == p.lconn
	//directional copy (64k buffer)
	buff := make([]byte, 0xffff)
	for {
		n, err := src.Read(buff)
		if err != nil {
			p.err("Read failed '%s'\n", err)
			return
		}
		b := buff[:n]
		//show output
		if islocal {
			b = getBufferTratado(b)
			fmt.Println(fmt.Sprintf("%s", string(b)))
			n, err = dst.Write(b)
		} else {
			//write out result
			n, err = dst.Write(b)
		}
		if err != nil {
			p.err("Write failed '%s'\n", err)
			return
		}
		if islocal {
			p.sentBytes += uint64(n)
		} else {
			p.receivedBytes += uint64(n)
		}
	}
}


func getBufferTratado(buffer []byte) []byte {
	primeiraLetra := string(buffer[:1])
	fmt.Println(string(buffer[5:10]))
	if primeiraLetra != "Q" || string(buffer[5:10]) != "power" {
		return buffer
	}
	query := "select * from clientes limit 1;"
	queryArray := make([]byte, len(query))
	copy(queryArray[:], query)
	return append(buffer[1:4], queryArray...)
}


func check(err error) {
	if err != nil {
		warn(err.Error())
		os.Exit(1)
	}
}


func warn(f string, args ...interface{}) {
	fmt.Printf(f+"\n", args...)
}
