package webapi

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var ConfigClient ClientConfig = ClientConfig{Port: 0, Params: make(map[string]interface{})}
var Cfg_srv_addr_map *map[string]interface{}

func ReadAppArgument(arg []string) {
	position := 1
	ConfigClient.Params["secret_service_address"] = ""
	ConfigClient.Params["secret_path"] = ""
	ConfigClient.Params["secret_service_port"] = 8200

	ConfigClient.Params["discovery_service_address"] = ""
	ConfigClient.Params["discovery_service_port"] = 8500
	ConfigClient.Params["check_health_interval"] = 10
	ConfigClient.Params["timeout"] = 30
	ConfigClient.Params["deregistry_delay_time"] = 60

	ConfigClient.Params["configuration_service_address"] = ""
	ConfigClient.Params["configuration_default_path"] = ""
	ConfigClient.Params["configuration_path"] = ""

	ConfigClient.Params["service_name"] = ""
	ConfigClient.Params["service_kind"] = ""
	ConfigClient.Params["service_port"] = 4000
	port := 0
	if len(arg) < 2 {
		log.Fatalf("Invalid command line arguments")
	}
	for {
		switch strings.ToLower(arg[position]) {
		case "help", "?":
			fmt.Println("./knb-run-action.exe -n|name name_value -m|mode mode_value -p|port port_value -r|registry r_params -v|vault v_params -c|config c_params")
			fmt.Println("\nname_value is the name of the microservice")
			fmt.Println("\nmode_value can be either 'action' or 'fetch'")
			fmt.Println("\nport_value is an integer value >=4000")
			fmt.Println("\nr_params is defined like this : server_address server_port health _interval timeout_interval quit_interval\n\tserver_address: servername or ip address of the server.\n\tserver_port: is the port value \n\thealth _interval: defines the periodical interval for checking the availability of the microservice\n\ttimeout_interval: is an interval of time where the microservice is supposed to response.\n\tquit_interval: an elapsed time where if the mocroservice does not response, the registry service delete it")
			fmt.Println("\nv_params is just the address of the vault server, it port and it secret path")
			fmt.Println("\nc_params is just the address of the configuration server and the default_path")
			fmt.Println("\n\tcopyright (c) 2023 ATOUBA Christian Lopez")
			os.Exit(0)
		case "-m", "-mode":
			if len(arg) > position+1 {
				ConfigClient.Params["service_kind"] = arg[position+1]
				position++
			}
		case "-p", "-port":
			if len(arg) > position+1 {
				port = 4000
				fmt.Sscanf(arg[position+1], "%d", &port)
				ConfigClient.Params["service_port"] = port
				position++
			}
		case "-r", "-registry":
			if len(arg) > position+1 {
				ConfigClient.Params["discovery_service_address"] = arg[position+1]
				position++
			}
			if len(arg) > position+1 {
				port = 8500
				fmt.Sscanf(arg[position+1], "%d", &port)
				ConfigClient.Params["discovery_service_port"] = port
				position++
			}
			if len(arg) > position+1 {
				port = 10
				fmt.Sscanf(arg[position+1], "%d", &port)
				ConfigClient.Params["check_health_interval"] = port
				position++
			}
			if len(arg) > position+1 {
				port = 30
				fmt.Sscanf(arg[position+1], "%d", &port)
				ConfigClient.Params["timeout"] = port
				position++
			}
			if len(arg) > position+1 {
				port = 60
				fmt.Sscanf(arg[position+1], "%d", &port)
				ConfigClient.Params["deregistry_delay_time"] = port
				position++
			}
		case "-v", "-vault":
			if len(arg) > position+1 {
				ConfigClient.Params["secret_service_address"] = arg[position+1]
				position++
			}
			if len(arg) > position+1 {
				port := 8200
				fmt.Sscanf(arg[position+1], "%d", &port)
				ConfigClient.Params["secret_service_port"] = port
				position++
			}
			if len(arg) > position+1 {
				ConfigClient.Params["secret_path"] = arg[position+1]
				position++
			}
		case "-c", "-config":
			if len(arg) > position+1 {
				ConfigClient.Params["configuration_service_address"] = arg[position+1]
				position++
			}
			if len(arg) > position+1 {
				ConfigClient.Params["configuration_default_path"] = arg[position+1]
				position++
			}
		case "-n", "-name":
			if len(arg) > position+1 {
				ConfigClient.Params["service_name"] = arg[position+1]
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
