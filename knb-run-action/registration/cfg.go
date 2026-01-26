package registration

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/akristianlopez/run-action/knb-run-action/webapi"
	"github.com/goccy/go-yaml"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
)

// var token string

func WatchConfig(addr, key string) (*webapi.Config, error) {
	// Create a watch plan for a specific KV key
	params := map[string]interface{}{
		"type": "key",
		"key":  key, // Change to your KV key
	}

	plan, err := watch.Parse(params)
	if err != nil {
		return nil, err
		// log.Fatalf("Error creating watch plan: %v", err)
	}

	// Handler function called when the key changes
	plan.Handler = func(idx uint64, data interface{}) {
		if data == nil {
			fmt.Println("Key deleted or not found")
			return
		}

		// KVPair type assertion
		if kv, ok := data.(*api.KVPair); ok {
			var conf webapi.Config
			if err := yaml.Unmarshal(kv.Value, &conf); err != nil {
				fmt.Printf("YAML invalide: %v", err)
				return
			}
			webapi.Db_connect_params = &webapi.Db_access_params{}
			webapi.Db_connect_params.Address = conf.Database.Address
			webapi.Db_connect_params.Port = int64(conf.Database.Port)
			webapi.Db_connect_params.Userid = conf.Database.Usrid
			webapi.Db_connect_params.Name = conf.Database.Name

			// // Load consul configuration
			// webapi.ConfigClient.Params["discovery_service_address"] = conf.Consul.URL
			// webapi.ConfigClient.Params["check_health_interval"] = conf.Consul.Health_check_interval
			// webapi.ConfigClient.Params["timeout"] = conf.Consul.Timeout
			// webapi.ConfigClient.Params["deregistry_delay_time"] = conf.Consul.Deregistry_delay_time

			// Load vault configuration
			webapi.ConfigClient.Params["secret_service_address"] = conf.Vault.URL
			webapi.ConfigClient.Params["secret_path"] = conf.Vault.Path
			webapi.ConfigClient.Params["secret_service_token"] = conf.Vault.Token

			webapi.Running_mode = webapi.ConfigClient.Params["service_kind"].(string)

			db_par, err := Read(fmt.Sprintf("http://%s", conf.Vault.URL), conf.Vault.Token, conf.Vault.Path /*"login", "password"*/) //"cubbyhole/webservice/db_access"
			if err != nil {
				log.Fatalf("Problem related to the database connection: '%s'", err.Error())
			}
			webapi.Db_connect_params.Password = db_par
			fmt.Printf("Change detected at index %d: %s = %s\n", idx, kv.Key, string(kv.Value))
		} else {
			fmt.Printf("Unexpected data type: %T\n", data)
		}
	}

	// Run the watch plan in a goroutine
	go func() {
		if err := plan.Run(addr); err != nil {
			log.Fatalf("Error running watch plan: %v", err)
		}
	}()

	// Graceful shutdown on Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("Stopping watch...")
	plan.Stop()
	os.Exit(0)
	return nil, nil
}

func ReadDefaultConfig(addr, key, ip string) {
	// Create a watch plan for a specific KV key
	// key := "wosa/default"
	if addr == "" || key == "" {
		log.Fatal("Configuration service's parameter is needed")
	}
	config := api.DefaultConfig()
	config.Address = addr
	client, err := api.NewClient(config)
	if err != nil {
		log.Fatalf("Error creating discovery client: %v", err)
	}

	// 2. Access the KV endpoint
	kv := client.KV()

	// 4. Read the key synchronously
	// The Get method blocks until it receives a response.
	kvPair /*queryMeta*/, _, err := kv.Get(key, nil)
	if err != nil {
		log.Fatalf("Error getting key %s: %v", key, err)
	}

	if kvPair == nil {
		log.Fatalf("Key %s not found", key)
	}
	// var conf webapi.DefaultConfig = webapi.DefaultConfig{}
	var conf map[string]interface{}

	if err := yaml.Unmarshal(kvPair.Value, &conf); err != nil {
		log.Fatalf("YAML invalide: %v", err)
	}
	webapi.ConfigClient.Params["service_name"] = (conf[ip].(map[string]interface{})["name"]).(string)
	webapi.ConfigClient.Params["service_port"] = (conf[ip].(map[string]interface{})["port"]).(uint64)
	webapi.ConfigClient.Params["configuration_path"] = (conf[ip].(map[string]interface{})["path"]).(string)
	webapi.ConfigClient.Params["service_kind"] = (conf[ip].(map[string]interface{})["mode"]).(string)
}
func ReadConfig(addr, key string) (*webapi.Config, error) {
	// Create a watch plan for a specific KV key
	// key := "wosa/default"
	if addr == "" || key == "" {
		log.Fatal("Configuration service's parameter is needed")
	}
	config := api.DefaultConfig()
	config.Address = addr
	client, err := api.NewClient(config)
	if err != nil {
		log.Fatalf("Error creating discovery client: %v", err)
	}

	// 2. Access the KV endpoint
	kv := client.KV()

	// 4. Read the key synchronously
	// The Get method blocks until it receives a response.
	kvPair /*queryMeta*/, _, err := kv.Get(key, nil)
	if err != nil {
		log.Fatalf("Error getting key %s: %v", key, err)
	}

	if kvPair == nil {
		log.Fatalf("Key %s not found", key)
	}
	// var conf webapi.DefaultConfig = webapi.DefaultConfig{}
	var conf webapi.Config
	if err := yaml.Unmarshal(kvPair.Value, &conf); err != nil {
		log.Fatalf("YAML invalide: %v", err)
	}

	return &conf, nil
}
