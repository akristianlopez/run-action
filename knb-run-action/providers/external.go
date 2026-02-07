package providers

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/akristianlopez/run-action/knb-run-action/registration"
	"github.com/akristianlopez/run-action/knb-run-action/webapi"
)

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
type StandAloneProvider struct {
	SysConf *webapi.ClientConfig
	conf    *webapi.Config
	ip      string
}

func (c *StandAloneProvider) registerService(cfg SwarmConsulConfig) error {
	return nil
}
func (c *StandAloneProvider) subscrib(url, topic, kind string) error {
	switch strings.ToLower(kind) {
	case "nats":
		webapi.Emit = registration.NatsPublish
		go registration.NatsSubscrib(url, topic)
		return nil
	case "kafka":
		webapi.Emit = registration.KafkaPublish
		go registration.KafkaSubscrib(url, topic)
		return nil
	case "rabbit":
		webapi.Emit = registration.RabbitMQPublish
		go registration.RabbitMQSubscrib(url, topic)
		return nil
	}
	return fmt.Errorf("Invalid kind value : %s", kind)
}
func (c *StandAloneProvider) ReadConfig() error {
	c.ip = GetLocalIP().String()
	// 1. lecture des arguments de la ligne de commandes
	c.readAppArgument(os.Args)

	name := os.Getenv("SERVICE_NAME")
	if name == "" {
		registration.ReadDefaultConfig(webapi.ConfigClient.Params["configuration_service_address"].(string),
			webapi.ConfigClient.Params["configuration_default_path"].(string), c.ip)
	} else {
		webapi.ConfigClient.Params["service_name"] = name
		port := 0
		fmt.Sscanf(os.Getenv("APP_PORT"), "%d", &port)
		webapi.ConfigClient.Params["service_port"] = port
		webapi.ConfigClient.Params["configuration_path"] = os.Getenv("WEBAPI_SRV_PATH")
		webapi.ConfigClient.Params["configuration_service_address"] = os.Getenv("WEBAPI_SRV_CONFIG_ADDRESS")
		webapi.ConfigClient.Params["service_kind"] = os.Getenv("WEBAPI_SRV_MODE")
	}
	if webapi.ConfigClient.Params["service_name"].(string) == "" {
		slog.Error("Missing service name")
		os.Exit(1)
	}
	if webapi.ConfigClient.Params["service_port"].(uint64) < 4000 {
		slog.Error("Invalid service port number. It must be greather than 4000")
		os.Exit(1)
	}
	if webapi.ConfigClient.Params["configuration_path"].(string) == "" {
		slog.Error("configuration path is required")
		os.Exit(1)
	}
	if webapi.ConfigClient.Params["service_kind"].(string) == "" {
		slog.Error("service mode is not defined")
		os.Exit(1)
	}
	if webapi.ConfigClient.Params["configuration_service_address"].(string) == "" {
		slog.Error("Missing configuration service address")
		os.Exit(1)
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
		slog.Error("Error while trying to read configuration", "error", err.Error())
		os.Exit(1)
	}
	if conf == nil {
		slog.Error("Configuration file is empty")
		os.Exit(1)
	}
	c.conf = conf
	// Load Database configuration
	// Except the password that will be retrieved from the vault server
	webapi.Db_connect_params = &webapi.Db_access_params{}
	webapi.Db_connect_params.Address = conf.Database.Address
	webapi.Db_connect_params.Port = int64(conf.Database.Port)
	webapi.Db_connect_params.Userid = conf.Database.Usrid
	webapi.Db_connect_params.Password = ""
	webapi.Db_connect_params.Name = conf.Database.Name
	webapi.Db_connect_params.Kind = conf.Database.Kind

	// Load consul configuration
	webapi.ConfigClient.Params["discovery_service_address"] = conf.Consul.URL
	webapi.ConfigClient.Params["check_health_interval"] = conf.Consul.Health_check_interval
	webapi.ConfigClient.Params["timeout"] = conf.Consul.Timeout
	webapi.ConfigClient.Params["deregistry_delay_time"] = conf.Consul.Deregistry_delay_time

	// Load vault configuration
	webapi.ConfigClient.Params["secret_service_address"] = conf.Vault.URL
	webapi.ConfigClient.Params["secret_path"] = conf.Vault.Path
	webapi.ConfigClient.Params["secret_service_token"] = conf.Vault.Token
	return nil
	// Reminds to treat :
	// - load kafka parameters
}

func (c *StandAloneProvider) readSecret(name string) (string, error) {
	return "", nil
}

func (c *StandAloneProvider) Launch() {
	IsDiscoveryServiceDefined := true
	IsVaultServiceDefined := true
	if webapi.ConfigClient.Params["discovery_service_address"].(string) == "" {
		slog.Info("Discovery service is missing")
		IsDiscoveryServiceDefined = false
	}
	if webapi.ConfigClient.Params["discovery_service_port"].(int) < 4000 {
		slog.Info("Discovery service port is not defined")
		IsDiscoveryServiceDefined = false
	}
	if webapi.ConfigClient.Params["check_health_interval"].(int) == 0 {
		slog.Info("Check health interval is not defined")
		IsDiscoveryServiceDefined = false
	}
	if webapi.ConfigClient.Params["timeout"].(int) == 0 {
		slog.Info("timeout value is not defined")
		IsDiscoveryServiceDefined = false
	}
	if webapi.ConfigClient.Params["deregistry_delay_time"].(int) == 0 {
		slog.Info("Deregistry delay value is not defined")
		IsDiscoveryServiceDefined = false
	}

	if webapi.ConfigClient.Params["secret_service_address"].(string) == "" {
		slog.Info("secret service url is not defined")
		IsVaultServiceDefined = false
	}
	if webapi.ConfigClient.Params["secret_path"].(string) == "" {
		slog.Info("secret path is not defined")
		IsVaultServiceDefined = false
	}
	if webapi.ConfigClient.Params["secret_service_token"].(string) == "" {
		slog.Info("connect token is not defined")
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

		if !CheckReachability(addr, fmt.Sprintf("%d", port),
			time.Duration(webapi.ConfigClient.Params["deregistry_delay_time"].(int))*time.Second) {
			slog.Error("Unreachable vault service 'http://%s:%d'",
				webapi.ConfigClient.Params["secret_service_address"].(string),
				webapi.ConfigClient.Params["secret_service_port"].(int))
			os.Exit(1)
		}

	}
	if webapi.ConfigClient.Params["service_kind"].(string) == "" {
		slog.Info("Warning: The mode value is not defined. We're going to fetch option as default value.")
		webapi.ConfigClient.Params["service_kind"] = "fetch"
	}

	if IsDiscoveryServiceDefined {
		tab := strings.Split(webapi.ConfigClient.Params["discovery_service_address"].(string), ":")
		addr := tab[0]
		port := 0
		if len(tab) > 1 {
			fmt.Sscanf(tab[len(tab)-1], "%d", &port)
		}
		if !CheckReachability(addr, fmt.Sprintf("%d", port),
			time.Duration(webapi.ConfigClient.Params["deregistry_delay_time"].(int))*time.Second) {
			slog.Error("Unreachable registry service 'http://%s:%d'", addr, port)
			os.Exit(1)
		}
		webapi.Running_mode = webapi.ConfigClient.Params["service_kind"].(string)

		db_par, err := registration.Read(fmt.Sprintf("http://%s", c.conf.Vault.URL), c.conf.Vault.Token, c.conf.Vault.Path /*"login", "password"*/) //"cubbyhole/webservice/db_access"
		if err != nil {
			slog.Error("Problem related to the database connection", "error", err.Error())
			os.Exit(1)
		}
		webapi.Db_connect_params.Password = db_par
		go registration.WatchConfig(
			webapi.ConfigClient.Params["configuration_service_address"].(string),
			webapi.ConfigClient.Params["configuration_path"].(string))
	}

	// Enregistrement du microservice dans consul
	//os.Hostname()
	if IsVaultServiceDefined && IsDiscoveryServiceDefined {
		n, e := registration.Register(webapi.ConfigClient.Params["service_port"].(uint64),
			webapi.ConfigClient.Params["service_name"].(string),

			webapi.ConfigClient.Params["discovery_service_address"].(string), c.ip,
			webapi.ConfigClient.Params["service_kind"].(string))
		if e != nil {
			slog.Error(e.Error())
			os.Exit(1)
		}
		webapi.Start(int(webapi.ConfigClient.Params["service_port"].(uint64))) //port a lire
		slog.Info(fmt.Sprintf("The microservice '%s' is running on port %d in '%s' mode", n, webapi.ConfigClient.Params["service_port"].(uint64),
			webapi.ConfigClient.Params["service_kind"].(string)))
		webapi.ReadSecret = nil

	}
	// Subscription to Nats jetstream topics
	if len(c.conf.Nats.Brokers) > 0 {
		webapi.Brokers = make([]webapi.BrokerInfo, len(c.conf.Nats.Brokers))
		copy(webapi.Brokers, c.conf.Nats.Brokers)
		for _, broker := range c.conf.Nats.Brokers {
			err := c.subscrib(broker.URL, broker.Topic, broker.Kind)
			if err != nil {
				slog.Error("Error while trying to subscrib", "error", err)
				os.Exit(1)
			}
		}
	}
}

func (c *StandAloneProvider) readAppArgument(arg []string) {
	position := 1
	webapi.ConfigClient.Params["secret_service_address"] = ""
	webapi.ConfigClient.Params["secret_path"] = ""
	webapi.ConfigClient.Params["secret_service_port"] = 8200

	webapi.ConfigClient.Params["discovery_service_address"] = ""
	webapi.ConfigClient.Params["discovery_service_port"] = 8500
	webapi.ConfigClient.Params["check_health_interval"] = 10
	webapi.ConfigClient.Params["timeout"] = 30
	webapi.ConfigClient.Params["deregistry_delay_time"] = 60

	webapi.ConfigClient.Params["configuration_service_address"] = ""
	webapi.ConfigClient.Params["configuration_default_path"] = ""
	webapi.ConfigClient.Params["configuration_path"] = ""

	webapi.ConfigClient.Params["service_name"] = ""
	webapi.ConfigClient.Params["service_kind"] = ""
	webapi.ConfigClient.Params["service_port"] = 4000
	port := 0
	if len(arg) < 2 {
		slog.Error("Invalid command line arguments")
		os.Exit(1)
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
				webapi.ConfigClient.Params["service_kind"] = arg[position+1]
				position++
			}
		case "-p", "-port":
			if len(arg) > position+1 {
				port = 4000
				fmt.Sscanf(arg[position+1], "%d", &port)
				webapi.ConfigClient.Params["service_port"] = port
				position++
			}
		case "-r", "-registry":
			if len(arg) > position+1 {
				webapi.ConfigClient.Params["discovery_service_address"] = arg[position+1]
				position++
			}
			if len(arg) > position+1 {
				port = 8500
				fmt.Sscanf(arg[position+1], "%d", &port)
				webapi.ConfigClient.Params["discovery_service_port"] = port
				position++
			}
			if len(arg) > position+1 {
				port = 10
				fmt.Sscanf(arg[position+1], "%d", &port)
				webapi.ConfigClient.Params["check_health_interval"] = port
				position++
			}
			if len(arg) > position+1 {
				port = 30
				fmt.Sscanf(arg[position+1], "%d", &port)
				webapi.ConfigClient.Params["timeout"] = port
				position++
			}
			if len(arg) > position+1 {
				port = 60
				fmt.Sscanf(arg[position+1], "%d", &port)
				webapi.ConfigClient.Params["deregistry_delay_time"] = port
				position++
			}
		case "-v", "-vault":
			if len(arg) > position+1 {
				webapi.ConfigClient.Params["secret_service_address"] = arg[position+1]
				position++
			}
			if len(arg) > position+1 {
				port := 8200
				fmt.Sscanf(arg[position+1], "%d", &port)
				webapi.ConfigClient.Params["secret_service_port"] = port
				position++
			}
			if len(arg) > position+1 {
				webapi.ConfigClient.Params["secret_path"] = arg[position+1]
				position++
			}
		case "-c", "-config":
			if len(arg) > position+1 {
				webapi.ConfigClient.Params["configuration_service_address"] = arg[position+1]
				position++
			}
			if len(arg) > position+1 {
				webapi.ConfigClient.Params["configuration_default_path"] = arg[position+1]
				position++
			}
		case "-n", "-name":
			if len(arg) > position+1 {
				webapi.ConfigClient.Params["service_name"] = arg[position+1]
				position++
			}
		default:
			slog.Error("Invalid command line arguments")
			os.Exit(1)
		}
		if position >= len(arg)-1 {
			break
		}
		position++
	}
}

func NewStandAloneProvider() ConfigProvider {
	return &StandAloneProvider{}
}
