// orbit-proxyd is the privileged Orbit edge daemon.
//
// It does two jobs:
//   - TCP forwards 127.0.0.2:80 -> 127.0.0.1:2080 (the unprivileged Orbit app).
//   - Runs a tiny authoritative DNS responder on 127.0.0.2:53 that answers
//     A queries for *.orbit.test with the alias IP. macOS routes only
//     `orbit.test` queries here via /etc/resolver/orbit.test, so Laravel Herd's
//     own dnsmasq on 127.0.0.1:53 keeps owning *.test untouched.
//
// Installed as a LaunchDaemon by scripts/setup-macos.sh.

package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	src := flag.String("src", "127.0.0.2:80", "tcp bind address")
	dst := flag.String("dst", "127.0.0.1:2080", "upstream address")
	tlsSrc := flag.String("tls-src", "", "tls bind address (e.g. 127.0.0.2:443); empty to disable")
	tlsDst := flag.String("tls-dst", "127.0.0.1:2443", "tls upstream address")
	dnsAddr := flag.String("dns", "127.0.0.2:53", "dns bind address")
	suffix := flag.String("suffix", "orbit.test", "domain suffix to answer")
	answer := flag.String("answer", "127.0.0.2", "A record IP to return")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("orbit-proxyd tcp %s -> %s | dns %s *.%s -> %s",
		*src, *dst, *dnsAddr, *suffix, *answer)
	if *tlsSrc != "" {
		log.Printf("orbit-proxyd tls %s -> %s", *tlsSrc, *tlsDst)
	}

	ip := net.ParseIP(*answer).To4()
	if ip == nil {
		log.Fatalf("answer must be IPv4: %s", *answer)
	}

	go runDNS(*dnsAddr, strings.ToLower(strings.TrimPrefix(*suffix, ".")), ip)

	if *tlsSrc != "" {
		go runForwarder(*tlsSrc, *tlsDst)
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
		<-sig
		os.Exit(0)
	}()

	runForwarder(*src, *dst)
}

func runForwarder(src, dst string) {
	ln, err := net.Listen("tcp", src)
	if err != nil {
		log.Fatalf("listen %s: %v", src, err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		go forward(conn, dst)
	}
}

func forward(client net.Conn, dst string) {
	defer client.Close()
	upstream, err := net.DialTimeout("tcp", dst, 3*time.Second)
	if err != nil {
		log.Printf("dial %s: %v", dst, err)
		return
	}
	defer upstream.Close()

	done := make(chan struct{}, 2)
	go pipe(upstream, client, done)
	go pipe(client, upstream, done)
	<-done
}

func pipe(dst, src net.Conn, done chan<- struct{}) {
	_, _ = io.Copy(dst, src)
	if tcp, ok := dst.(*net.TCPConn); ok {
		_ = tcp.CloseWrite()
	}
	done <- struct{}{}
}

// runDNS serves UDP DNS on addr, answering A queries whose qname equals
// suffix or ends in "."+suffix with ip. Anything else gets NXDOMAIN.
func runDNS(addr, suffix string, ip net.IP) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatalf("dns resolve %s: %v", addr, err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("dns listen %s: %v", addr, err)
	}
	log.Printf("dns listening on %s", addr)

	buf := make([]byte, 1500)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("dns read: %v", err)
			continue
		}
		resp, ok := buildDNSResponse(buf[:n], suffix, ip)
		if !ok {
			continue
		}
		if _, err := conn.WriteToUDP(resp, src); err != nil {
			log.Printf("dns write: %v", err)
		}
	}
}

const (
	dnsTypeA   = 1
	dnsClassIN = 1
)

// buildDNSResponse parses a query and crafts an authoritative reply.
// Returns ok=false on malformed input.
func buildDNSResponse(req []byte, suffix string, ip net.IP) ([]byte, bool) {
	if len(req) < 12 {
		return nil, false
	}
	qdcount := int(req[4])<<8 | int(req[5])
	if qdcount != 1 {
		return nxdomain(req), true
	}

	qname, qtype, qclass, qend, ok := parseQuestion(req, 12)
	if !ok {
		return nil, false
	}

	resp := make([]byte, qend, qend+16)
	copy(resp, req[:qend])
	// flags: QR=1, AA=1, RD copied from request, RA=0, RCODE=0
	resp[2] = 0x84 | (req[2] & 0x01)
	resp[3] = 0x00
	// ancount = 1 if we answer, else NXDOMAIN
	matches := qtype == dnsTypeA && qclass == dnsClassIN && nameMatches(qname, suffix)
	if !matches {
		resp[3] = 0x03 // NXDOMAIN
		setCounts(resp, 1, 0, 0, 0)
		return resp, true
	}
	setCounts(resp, 1, 1, 0, 0)

	// Answer: name pointer to offset 12, TYPE A, CLASS IN, TTL 60, RDLEN 4, RDATA ip
	ans := []byte{
		0xC0, 0x0C,
		0x00, dnsTypeA,
		0x00, dnsClassIN,
		0x00, 0x00, 0x00, 0x3C,
		0x00, 0x04,
		ip[0], ip[1], ip[2], ip[3],
	}
	resp = append(resp, ans...)
	return resp, true
}

func setCounts(resp []byte, qd, an, ns, ar int) {
	resp[4], resp[5] = byte(qd>>8), byte(qd)
	resp[6], resp[7] = byte(an>>8), byte(an)
	resp[8], resp[9] = byte(ns>>8), byte(ns)
	resp[10], resp[11] = byte(ar>>8), byte(ar)
}

func nxdomain(req []byte) []byte {
	resp := make([]byte, 12)
	copy(resp, req[:12])
	resp[2] = 0x84 | (req[2] & 0x01)
	resp[3] = 0x03
	setCounts(resp, 0, 0, 0, 0)
	return resp
}

func parseQuestion(req []byte, off int) (name string, qtype, qclass, end int, ok bool) {
	var labels []string
	for off < len(req) {
		l := int(req[off])
		if l == 0 {
			off++
			break
		}
		if l&0xC0 != 0 {
			return "", 0, 0, 0, false
		}
		off++
		if off+l > len(req) {
			return "", 0, 0, 0, false
		}
		labels = append(labels, strings.ToLower(string(req[off:off+l])))
		off += l
	}
	if off+4 > len(req) {
		return "", 0, 0, 0, false
	}
	qtype = int(req[off])<<8 | int(req[off+1])
	qclass = int(req[off+2])<<8 | int(req[off+3])
	return strings.Join(labels, "."), qtype, qclass, off + 4, true
}

func nameMatches(qname, suffix string) bool {
	return qname == suffix || strings.HasSuffix(qname, "."+suffix)
}
