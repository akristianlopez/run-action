package main

/*
	This microservice is built to execute the action within a specific database.
	this microservice register itself into a consul service registry and
	retrieve the database connection parameters from a vault server.
	it exposes four main functionalities:
		- run action: execute an action and return the result;
		- fetch data: fetch data from the database and return the result;
		- check: check if a specific action can be executed or if a specific expression can be parsed on a specific table
		  whose the name is given;
		- ping: health check of the microservice.
	This service requires the following command line arguments:
		- mode: defines the running mode of the microservice. It can be either 'action' or 'fetch';
		- port: defines the port number where the microservice is listening	;
		- registry: defines the address and port of the consul service registry as well as the check interval, timeout interval
		  and quit interval;
		- vault: defines the address and port of the vault server;
		- config: defines the address of the configuration service and the default configuration path;
		- name: the name of the microservice to be registered into the consul service registry.
	This microservice requires also the following environment variables:
		- WEBAPI_VAULT_TOKEN: the token used to access the vault server;
		- WEBAPI_SRV_NAME: the name of the microservice to be registered into the consul service registry.
*/

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
func getLocalIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}
func main() {
	/*
	   Structure de la ligne de commande
	   ./nom_exe ?|help
	   ./nom_exe -r|registry server_address check_interval timeout_interval quit_interval
	   ./nom_exe -v|vault server_address
	   ./nom_exe -p|port port
	   ./nom_exe -m|mode role ('fetch','action')
	   ./nom_exe -c|config addr config_path_default
	   ./nom_exe -n|name name_service
	   Les differents parametres peuvent etre donnes dans n'importe quel ordre.
	   Les parametres peuvent etre donnes de maniere individuelle ou combinee. Par exemple:
	   ./nom_exe -r addr 10 5 5 -v addr -p 5000 -m fetch -c addr path -n service_name
	   ./nom_exe -r addr 10 5 5 -v addr -p 5000 -m fetch -c addr path
	   ./nom_exe -p 5000 -m fetch -n service_name
	   ./nom_exe -m fetch -p 5000
	   ces differentes options peuvent etre combinees.
	*/
	webapi.ReadAppArgument(os.Args)
	name := os.Getenv("WEBAPI_SRV_NAME")
	ip := getLocalIP()

	if name == "" {
		registration.ReadDefaultConfig(webapi.ConfigClient.Params["configuration_service_address"].(string),
			webapi.ConfigClient.Params["configuration_default_path"].(string), ip.String())
	} else {
		webapi.ConfigClient.Params["service_name"] = name
		port := 0
		fmt.Sscanf(os.Getenv("WEBAPI_SRV_PORT"), "%d", &port)
		webapi.ConfigClient.Params["service_port"] = port
		webapi.ConfigClient.Params["configuration_path"] = os.Getenv("WEBAPI_SRV_PATH")
		webapi.ConfigClient.Params["configuration_service_address"] = os.Getenv("WEBAPI_SRV_CONFIG_ADDRESS")
		webapi.ConfigClient.Params["service_kind"] = os.Getenv("WEBAPI_SRV_MODE")
	}
	if webapi.ConfigClient.Params["service_name"].(string) == "" {
		log.Fatal("Missing service name")
	}
	if webapi.ConfigClient.Params["service_port"].(uint64) < 4000 {
		log.Fatal("Invalid service port number. It must be greather than 4000")
	}
	if webapi.ConfigClient.Params["configuration_path"].(string) == "" {
		log.Fatal("configuration path is required")
	}
	if webapi.ConfigClient.Params["service_kind"].(string) == "" {
		log.Fatal("service mode is not defined")
	}
	if webapi.ConfigClient.Params["configuration_service_address"].(string) == "" {
		log.Fatal("Missing configuration service address")
	}

	os.Setenv("WEBAPI_SRV_NAME", webapi.ConfigClient.Params["service_name"].(string))
	os.Setenv("WEBAPI_SRV_PORT", fmt.Sprintf("%d", webapi.ConfigClient.Params["service_port"].(uint64)))
	os.Setenv("WEBAPI_SRV_PATH", webapi.ConfigClient.Params["configuration_path"].(string))
	os.Setenv("WEBAPI_SRV_CONFIG_ADDRESS", webapi.ConfigClient.Params["configuration_service_address"].(string))
	os.Setenv("WEBAPI_SRV_MODE", webapi.ConfigClient.Params["service_kind"].(string))
	webapi.ExistingService = registration.ExistingService
	webapi.IsServiceExists = registration.IsServiceExists

	conf, err := registration.ReadConfig(webapi.ConfigClient.Params["configuration_service_address"].(string), webapi.ConfigClient.Params["configuration_path"].(string))
	if err != nil {
		log.Fatalf("Error while trying to read configuration: %s", err.Error())
	}
	if conf == nil {
		log.Fatal("Configuration file is empty")
	}
	// Load Database configuration
	// Except the password that will be retrieved from the vault server
	webapi.Db_connect_params = &webapi.Db_access_params{}
	webapi.Db_connect_params.Address = conf.Database.Address
	webapi.Db_connect_params.Port = int64(conf.Database.Port)
	webapi.Db_connect_params.Userid = conf.Database.Usrid
	webapi.Db_connect_params.Password = ""
	webapi.Db_connect_params.Name = conf.Database.Name

	// Load consul configuration
	webapi.ConfigClient.Params["discovery_service_address"] = conf.Consul.URL
	webapi.ConfigClient.Params["check_health_interval"] = conf.Consul.Health_check_interval
	webapi.ConfigClient.Params["timeout"] = conf.Consul.Timeout
	webapi.ConfigClient.Params["deregistry_delay_time"] = conf.Consul.Deregistry_delay_time

	// Load vault configuration
	webapi.ConfigClient.Params["secret_service_address"] = conf.Vault.URL
	webapi.ConfigClient.Params["secret_path"] = conf.Vault.Path
	webapi.ConfigClient.Params["secret_service_token"] = conf.Vault.Token

	// Reminds to treat :
	// - load kafka parameters
	IsDiscoveryServiceDefined := true
	IsVaultServiceDefined := true
	if webapi.ConfigClient.Params["discovery_service_address"].(string) == "" {
		fmt.Println("Discovery service is missing")
		IsDiscoveryServiceDefined = false
	}
	if webapi.ConfigClient.Params["discovery_service_port"].(int) < 4000 {
		fmt.Println("Discovery service port is not defined")
		IsDiscoveryServiceDefined = false
	}
	if webapi.ConfigClient.Params["check_health_interval"].(int) == 0 {
		fmt.Println("Check health interval is not defined")
		IsDiscoveryServiceDefined = false
	}
	if webapi.ConfigClient.Params["timeout"].(int) == 0 {
		fmt.Println("timeout value is not defined")
		IsDiscoveryServiceDefined = false
	}
	if webapi.ConfigClient.Params["deregistry_delay_time"].(int) == 0 {
		fmt.Println("Deregistry delay value is not defined")
		IsDiscoveryServiceDefined = false
	}

	if webapi.ConfigClient.Params["secret_service_address"].(string) == "" {
		fmt.Println("secret service url is not defined")
		IsVaultServiceDefined = false
	}
	if webapi.ConfigClient.Params["secret_path"].(string) == "" {
		fmt.Println("secret path is not defined")
		IsVaultServiceDefined = false
	}
	if webapi.ConfigClient.Params["secret_service_token"].(string) == "" {
		fmt.Println("connect token is not defined")
		IsVaultServiceDefined = false
	}

	// Connexion Consul
	// if !validateIP(webapi.ConfigClient.Params["discovery_service_address"].(string)) &&
	// 	!validateHostname(webapi.ConfigClient.Params["discovery_service_address"].(string)) {
	// 	log.Fatal("Invalid consul server address")
	// }
	// if !validateIP(webapi.ConfigClient.Params["secret_service_address"].(string)) &&
	// 	!validateHostname(webapi.ConfigClient.Params["secret_service_address"].(string)) {
	// 	log.Fatal("Invalid vault server address")
	// }
	if IsVaultServiceDefined {
		tab := strings.Split(webapi.ConfigClient.Params["secret_service_address"].(string), ":")
		addr := tab[0]
		port := 0
		if len(tab) > 1 {
			fmt.Sscanf(tab[len(tab)-1], "%d", &port)
		}

		if !checkReachability(addr, fmt.Sprintf("%d", port),
			time.Duration(webapi.ConfigClient.Params["deregistry_delay_time"].(int))*time.Second) {
			log.Fatalf("Unreachable vault service 'http://%s:%d'",
				webapi.ConfigClient.Params["secret_service_address"].(string),
				webapi.ConfigClient.Params["secret_service_port"].(int))
		}

	}
	if webapi.ConfigClient.Params["service_kind"].(string) == "" {
		fmt.Println("Warning: The mode value is not defined. We're going to fetch option as default value.")
		webapi.ConfigClient.Params["service_kind"] = "fetch"
	}

	if IsDiscoveryServiceDefined {
		tab := strings.Split(webapi.ConfigClient.Params["discovery_service_address"].(string), ":")
		addr := tab[0]
		port := 0
		if len(tab) > 1 {
			fmt.Sscanf(tab[len(tab)-1], "%d", &port)
		}
		if !checkReachability(addr, fmt.Sprintf("%d", port),
			time.Duration(webapi.ConfigClient.Params["deregistry_delay_time"].(int))*time.Second) {
			log.Fatalf("Unreachable registry service 'http://%s:%d'", addr, port)
		}
		webapi.Running_mode = webapi.ConfigClient.Params["service_kind"].(string)

		db_par, err := registration.Read(fmt.Sprintf("http://%s", conf.Vault.URL), conf.Vault.Token, conf.Vault.Path /*"login", "password"*/) //"cubbyhole/webservice/db_access"
		if err != nil {
			log.Fatalf("Problem related to the database connection: '%s'", err.Error())
		}
		webapi.Db_connect_params.Password = db_par
		go func() {
			registration.WatchConfig(
				webapi.ConfigClient.Params["configuration_service_address"].(string),
				webapi.ConfigClient.Params["configuration_path"].(string))
		}()
	}

	// Enregistrement du microservice dans consul
	//os.Hostname()
	if IsVaultServiceDefined && IsDiscoveryServiceDefined {
		n, e := registration.Register(webapi.ConfigClient.Params["service_port"].(uint64),
			webapi.ConfigClient.Params["service_name"].(string),

			webapi.ConfigClient.Params["discovery_service_address"].(string), ip.String(),
			webapi.ConfigClient.Params["service_kind"].(string))
		if e != nil {
			log.Fatal(e.Error())
		}
		webapi.Start(int(webapi.ConfigClient.Params["service_port"].(uint64))) //port a lire
		log.Printf("The microservice '%s' is running on port %d in '%s' mode", n, webapi.ConfigClient.Params["service_port"].(uint64),
			webapi.ConfigClient.Params["service_kind"].(string))
	}

}
