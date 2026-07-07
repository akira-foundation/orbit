package share

import (
	"errors"
	"net"
)

func PrimaryLANIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if v4 := ip.To4(); v4 != nil {
				return v4.String(), nil
			}
		}
	}
	return "", errors.New("share: no non-loopback IPv4 interface")
}

func ShareURL(ip, port, slug string) string {
	return "http://" + ip + ":" + port + "/?__orbit=" + slug
}
