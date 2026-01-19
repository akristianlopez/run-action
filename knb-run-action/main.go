package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/akristianlopez/run-action/knb-run-action/registration"
	"github.com/akristianlopez/run-action/knb-run-action/webapi"
)

var (
	v_server_addr, reg_server_addr, mode                string
	port, chk_interval, to_interval, q_interval, v_port int
)

func readAppArgument(arg []string) {
	position := 1
	v_server_addr = ""
	reg_server_addr = ""
	mode = ""
	port = 4000
	chk_interval = 10
	to_interval = 30
	q_interval = 60
	if len(arg) < 2 {
		log.Fatalf("Invalid command line arguments")
	}
	for {
		switch strings.ToLower(arg[position]) {
		case "help", "?":
			fmt.Println("./knb-run-action.exe -m|mode mode_value -p|port port_value -r|registry r_params -v|vault v_params")
			fmt.Println("mode_value can be either 'action' or 'fetch'")
			fmt.Println("port_value is an integer value >=4000")
			fmt.Println("r_params is defined like this : \n\tserver_address servername or ip address of the server.\n\thealth _interval defines the periodical interval for checking the availability of the microservice\n\ttimeout_interval is interval of time where the microservice is supposed to response.\n\tquit_interval an lap of time where if the mocroservice does not response, the service delete it")
			fmt.Println("v_params is just the address of the vault server")
			fmt.Println("\tcopyright (c) 2023 ATOUBA Christian Lopez")
			os.Exit(0)
		case "-m", "-mode":
			if len(arg) > position+1 {
				mode = arg[position+1]
				position++
			}
		case "-p", "-port":
			if len(arg) > position+1 {
				fmt.Sscanf(arg[position+1], "%d", &port)
				position++
			}
		case "-r", "-registry":
			if len(arg) > position+1 {
				reg_server_addr = arg[position+1]
				position++
			}
			if len(arg) > position+1 {
				fmt.Sscanf(arg[position+1], "%d", &chk_interval)
				position++
			}
			if len(arg) > position+1 {
				fmt.Sscanf(arg[position+1], "%d", &to_interval)
				position++
			}
			if len(arg) > position+1 {
				fmt.Sscanf(arg[position+1], "%d", &q_interval)
				position++
			}
		case "-v", "-vault":
			if len(arg) > position+1 {
				v_server_addr = arg[position+1]
				position++
			}
			if len(arg) > position+1 {
				fmt.Sscanf(arg[position+1], "%d", &v_port)
				position++
			}
		default:
			log.Fatalf("Invalid command line arguments")
			return
		}
		if position >= len(arg)-1 {
			break
		}
		position++
	}
}

// validateIP checks if the given string is a valid IPv4 or IPv6 address
func validateIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// validateHostname checks if the hostname is syntactically valid
func validateHostname(host string) bool {
	// RFC 1123: Hostname can contain letters, digits, hyphens, and dots
	// Each label must be 1-63 chars, total length <= 253
	hostnameRegex := `^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)*` +
		`[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`
	re := regexp.MustCompile(hostnameRegex)
	return len(host) <= 253 && re.MatchString(host)
}

// checkReachability tries to connect to the host on a given port
func checkReachability(host string, port string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
func main() {
	/*
	   Structure de la ligne de commande
	   ./nom_exe ?|help
	   ./nom_exe -r|registry server_address check_interval timeout_interval quit_interval
	   ./nom_exe -v|vault server_address
	   ./nom_exe -p port
	   ./nom_exe -m role ('fetch','action')
	   ces differentes options peuvent etre combinees.
	*/
	readAppArgument(os.Args)

	fmt.Println("Address : ", reg_server_addr)
	fmt.Println("Check_interval : ", chk_interval)
	fmt.Println("Timeout_interval : ", to_interval)
	fmt.Println("Quit_interval :", q_interval)
	fmt.Println("Mode :", mode)
	fmt.Println("Port :", port)
	fmt.Println("Vault server: ", reg_server_addr)
	fmt.Println("Vault port : ", v_port)

	if !validateIP(reg_server_addr) && !validateHostname(reg_server_addr) {
		log.Fatalf("Invalid consul server address")
	}
	if !validateIP(v_server_addr) && !validateHostname(v_server_addr) {
		log.Fatalf("Invalid vault server address")
	}
	if mode == "" {
		fmt.Println("Warning: The mode value is not defined. We're going to fetch option as default value.")
		mode = "fetch"
	}
	webapi.Running_mode = mode

	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		log.Fatalf("'VAULT_TOKEN' token is not defined")
	}
	db_par, err := registration.Read(fmt.Sprintf("%s:%d", v_server_addr, v_port), token, "secret/data", "db_access" /*"login", "password"*/)
	if err != nil {
		log.Fatalf("Problem related to the database connection: '%s'", err.Error())
	}

	// Enregistrement du microservice dans consul
	//os.Hostname()
	n, _ := registration.Register(port, reg_server_addr, mode)
	webapi.Db_connect_params = db_par
	webapi.Start(port) //port a lire
	// registration.Write("http://127.0.0.1:8200", token, "secret/data/ma-cle-secrete" /*"login", "password",*/, "nouvelle_valeur")
	log.Printf("The registration of the Serice '%s' was succeeded", n)

}
