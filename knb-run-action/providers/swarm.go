package providers

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/akristianlopez/run-action/knb-run-action/registration"
	"github.com/akristianlopez/run-action/knb-run-action/webapi"
	"github.com/goccy/go-yaml"
)

type SwarmProvider struct {
	Secret_key string
	svcConf    *SwarmConsulConfig
	SysConf    *webapi.ConfigEx
}

func (c *SwarmProvider) ReadConfig() error {
	// 1. Loading the technical configuration
	cfg := c.loadBootstrapConfig()

	// 2. Reading JWt signature secret (JWT key)
	jwtKey, err := c.readSecret("jwt_key")
	if err != nil {
		slog.Error("❌ Erreur fatale : Signature key for JWT not found", "error", err)
		os.Exit(1)
	}
	slog.Info("🔐 JWT secret key loaded successfully")
	webapi.ConfigClient.Params["jwt_key"] = jwtKey
	c.Secret_key = jwtKey
	c.svcConf = &cfg
	return nil
}
func (c *SwarmProvider) loadBootstrapConfig() SwarmConsulConfig {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	return SwarmConsulConfig{
		Port:        port,
		ServiceName: os.Getenv("SERVICE_NAME"),
		ConsulAddr:  os.Getenv("CONSUL_HTTP_ADDR"),
	}
}

// readSecret lit un fichier dans /run/secrets/
func (c *SwarmProvider) readSecret(name string) (string, error) {
	path := "/run/secrets/" + name
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func (c *SwarmProvider) registerService(scf SwarmConsulConfig) error {
	host_name, err := os.Hostname()
	if err != nil {
		slog.Error("Fatal error related to the Hostname", "error", err)
		os.Exit(1)
	}
	conf, err := c.loadConfig()
	if err != nil {
		return err
	}
	c.SysConf = conf

	err = registration.RegisterEx(scf.Port, scf.ServiceName, scf.ConsulAddr, host_name)
	if err == nil {
		p := 0
		fmt.Sscanf(scf.Port, "%d", &p)
		webapi.ConfigClient.Params["service_port"] = p
		webapi.ConfigClient.Params["service_kind"] = "swarm"
		webapi.ConfigClient.Params["service_name"] = scf.ServiceName
		webapi.ConfigClient.Params["discovery_service_address"] = scf.ConsulAddr
	}
	return err
}
func (c *SwarmProvider) subscrib(url, topic string) error {
	return registration.Subscrib(url, topic)
}
func (c *SwarmProvider) Launch() {
	pwd, err := c.readSecret("db_config")
	if err != nil {
		slog.Error(fmt.Sprintf("Invalid config file name: %s", pwd))
		os.Exit(1)
	}
	webapi.Emit = registration.Emit

	// 1. Starts the knb service
	webapi.Start(int(webapi.ConfigClient.Params["service_port"].(uint64))) //port a lire

	// 2. Register knb service to the consul service
	go c.registerService(*c.svcConf)
	webapi.ReadSecret = c.readSecret
	// 3. Subscrib to the topics from broker
	if len(c.SysConf.Nats.Brokers) > 0 {
		webapi.Brokers = make([]webapi.BrokerInfo, len(c.SysConf.Nats.Brokers))
		copy(webapi.Brokers, c.SysConf.Nats.Brokers)
		for _, broker := range webapi.Brokers {
			go c.subscrib(broker.URL, broker.Topic)
		}
	}
	slog.Info("The microservice is running",
		"service_name", c.svcConf.ServiceName,
		"service_port", webapi.ConfigClient.Params["service_port"].(uint64),
		"service_kind", webapi.ConfigClient.Params["service_kind"].(string))
	// le fichier de configuration doit comporter les information sur le brokers
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	hn, _ := os.Hostname()

	// Desenregister the microservice from consul
	err = registration.Deregister(hn, webapi.ConfigClient.Params["discovery_service_address"].(string))
	if err != nil {
		slog.Info(err.Error())
	}

	// Wait 20 secondes after deregistration before closing the microservice
	time.Sleep(time.Duration(webapi.Deregister_waiting_time) * time.Second)
	slog.Info("Stopping watch...")

	os.Exit(0)
}

func (c *SwarmProvider) loadConfig() (*webapi.ConfigEx, error) {
	// 1. Chemin par défaut (celui défini dans le volume/config du YAML)
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/run/config.yaml"
	}

	// 2. Lecture du fichier
	data, err := os.ReadFile(configPath)
	if err != nil {
		slog.Info("⚠️ Aucun fichier de config trouvé, utilisation des envs", "configPath", configPath)
		return nil, err // Fallback
	}

	// 3. Parsing (nécessite une lib comme gopkg.in/yaml.v3)
	var cfg webapi.ConfigEx
	yaml.Unmarshal(data, &cfg)
	return &cfg, nil
}

func NewSwarmProvider() ConfigProvider {
	return &SwarmProvider{}
}
