package providers

import (
	"log"
	"net"
	"regexp"
	"time"
)

type ConfigProvider interface {
	// GetSecret(name string) (string, error)
	registerService(c SwarmConsulConfig) error
	ReadConfig() error
	subscrib(url, topic, kind string) error
	readSecret(name string) (string, error)
	Launch()
}
type SwarmConsulConfig struct {
	Port        string
	ServiceName string
	ConsulAddr  string
}

// validateIP checks if the given string is a valid IPv4 or IPv6 address

func ValidateIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// validateHostname checks if the hostname is syntactically valid
func ValidateHostname(host string) bool {
	// RFC 1123: Hostname can contain letters, digits, hyphens, and dots
	// Each label must be 1-63 chars, total length <= 253
	hostnameRegex := `^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)*` +
		`[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`
	re := regexp.MustCompile(hostnameRegex)
	return len(host) <= 253 && re.MatchString(host)
}

// checkReachability tries to connect to the host on a given port
func CheckReachability(host string, port string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
func GetLocalIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}
