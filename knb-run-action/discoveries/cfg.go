package discoveries

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	secrets "github.com/akristianlopez/run-action/knb-run-action/secrets/vault"
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
			slog.Info("Key deleted or not found")
			return
		}

		// KVPair type assertion
		if kv, ok := data.(*api.KVPair); ok {
			var conf webapi.Config
			if err := yaml.Unmarshal(kv.Value, &conf); err != nil {
				slog.Warn("YAML invalide", "error", err)
				return
			}
			db_par, err := secrets.Read(fmt.Sprintf("http://%s", conf.Vault.URL), conf.Vault.Token, conf.Vault.Path /*"login", "password"*/) //"cubbyhole/webservice/db_access"
			if err != nil {
				slog.Error("Problem related to the database connection", "error", err.Error())
				os.Exit(1)
			}

			if webapi.Db_connect_params == nil {
				webapi.Db_connect_params = &webapi.Db_access_params{}
			}
			webapi.Db_connect_params.Address = conf.Database.Address
			webapi.Db_connect_params.Port = int64(conf.Database.Port)
			webapi.Db_connect_params.Userid = conf.Database.Usrid
			webapi.Db_connect_params.Name = conf.Database.Name
			webapi.Db_connect_params.Kind = conf.Database.Kind
			webapi.Db_connect_params.Password = db_par
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

			slog.Info(fmt.Sprintf("Change detected at index %d: %s = %s\n", idx, kv.Key, conf.Database.Address))
		} else {
			slog.Warn("Unexpected data type", "data", data)
		}
	}

	// Run the watch plan in a goroutine
	go func() {
		if err := plan.Run(addr); err != nil {
			slog.Error("Error running watch plan", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown on Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	hn, _ := os.Hostname()
	// Deregister the microservice from consul
	err = Deregister(hn, webapi.ConfigClient.Params["discovery_service_address"].(string))
	if err != nil {
		slog.Error(err.Error())
	}
	plan.Stop()
	time.Sleep(time.Duration(webapi.Deregister_waiting_time) * time.Second)
	slog.Info("Stopping watch...")

	// Wait 20 secondes after deregistration before closing the microservice

	// thinking about stopping the service without abort current task
	os.Exit(0)
	return nil, nil
}

func ReadDefaultConfig(addr, key, ip string) {
	// Create a watch plan for a specific KV key
	// key := "wosa/default"
	if addr == "" || key == "" {
		slog.Error("Configuration service's parameter is needed")
		os.Exit(1)
	}
	config := api.DefaultConfig()
	config.Address = addr
	client, err := api.NewClient(config)
	if err != nil {
		slog.Error("Error creating discovery client", "error", err)
		os.Exit(1)
	}

	// 2. Access the KV endpoint
	kv := client.KV()

	// 4. Read the key synchronously
	// The Get method blocks until it receives a response.
	kvPair /*queryMeta*/, _, err := kv.Get(key, nil)
	if err != nil {
		slog.Error(fmt.Sprintf("Error getting key %s: %v", key, err))
		os.Exit(1)
	}

	if kvPair == nil {
		slog.Error(fmt.Sprintf("Key %s not found", key))
		os.Exit(1)
	}
	// var conf webapi.DefaultConfig = webapi.DefaultConfig{}
	var conf map[string]interface{}

	if err := yaml.Unmarshal(kvPair.Value, &conf); err != nil {
		slog.Error(fmt.Sprintf("YAML invalide: %v", err))
		os.Exit(1)
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
		slog.Error("Configuration service's parameter is needed")
		os.Exit(1)
	}
	config := api.DefaultConfig()
	config.Address = addr
	client, err := api.NewClient(config)
	if err != nil {
		slog.Error(fmt.Sprintf("Error creating discovery client: %v", err))
		os.Exit(1)
	}

	// 2. Access the KV endpoint
	kv := client.KV()

	// 4. Read the key synchronously
	// The Get method blocks until it receives a response.
	kvPair /*queryMeta*/, _, err := kv.Get(key, nil)
	if err != nil {
		slog.Error(fmt.Sprintf("Error getting key %s: %v", key, err))
		os.Exit(1)
	}

	if kvPair == nil {
		slog.Error(fmt.Sprintf("Key %s not found", key))
		os.Exit(1)
	}
	// var conf webapi.DefaultConfig = webapi.DefaultConfig{}
	var conf webapi.Config
	if err := yaml.Unmarshal(kvPair.Value, &conf); err != nil {
		slog.Error(fmt.Sprintf("YAML invalide: %v", err))
		os.Exit(1)
	}

	return &conf, nil
}
