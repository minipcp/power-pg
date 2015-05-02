package proxy

import (
	"github.com/minipcp/power-pg/common"
	"fmt"
	"net"
	"os"
	"io"
	"encoding/binary"
)


var (
	connid = uint64(0)
)

func Start(localHost, remoteHost *string, powerCallback common.Callback) {
	fmt.Printf("Proxying from %v to %v\n", localHost, remoteHost)

	localAddr, remoteAddr := getResolvedAddresses(localHost, remoteHost)
	listener := getListener(localAddr)

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			fmt.Printf("Failed to accept connection '%s'\n", err)
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
		go p.start(powerCallback)
	}
}


func getResolvedAddresses(localHost, remoteHost *string) (*net.TCPAddr, *net.TCPAddr) {
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


func (p *proxy) start(powerCallback common.Callback) {
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
	go p.pipe(p.lconn, p.rconn, powerCallback)
	go p.pipe(p.rconn, p.lconn, nil)
	//wait for close...
	<-p.errsig
}


func (p *proxy) pipe(src, dst *net.TCPConn, powerCallback common.Callback) {
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
			b = getModifiedBuffer(b, powerCallback)
			n, err = dst.Write(b)
		} else {
			//write out result
			n, err = dst.Write(b)
		}
		if err != nil {
			p.err("Write failed '%s'\n", err)
			return
		}
	}
}


func getModifiedBuffer(buffer []byte, powerCallback common.Callback) []byte {
	if powerCallback == nil || len(buffer) < 1 || string(buffer[0]) != "Q" || string(buffer[5:11]) != "power:" {
		return buffer
	}
	query := powerCallback(string(buffer[5:]))
	return makeMessage(query)
}


func makeMessage(query string) []byte {
	queryArray := make([]byte, 0, 6+len(query))
	queryArray = append(queryArray, 'Q', 0, 0, 0, 0)
	queryArray = append(queryArray, query...)
	queryArray = append(queryArray, 0)
	binary.BigEndian.PutUint32(queryArray[1:], uint32(len(queryArray)-1))
	return queryArray

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

