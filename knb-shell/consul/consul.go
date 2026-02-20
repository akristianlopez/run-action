package consul

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/hashicorp/consul/api"
	"gopkg.in/yaml.v2"
)

// RemoteModule represents a discovered micro-frontend
type RemoteModule struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Client encapsulates interactions with Consul
type Client struct {
	api    *api.Client
	logger *slog.Logger
}
type Labels struct {
	Tags []string `yaml:"tags" json:"tags"`
}

// NewClient initializes a connection with the Consul agent
func NewClient(address string, logger *slog.Logger) (*Client, error) {
	config := api.DefaultConfig()
	config.Address = address
	apiClient, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &Client{api: apiClient, logger: logger}, nil
}

// Register declares the service in Consul with Traefik tags
func (c *Client) Register(name, id string, port int, tags *Labels) error {
	serviceKind := os.Getenv("SERVICE_KIND")
	url := GetLocalIP().String()
	if strings.EqualFold(serviceKind, "swarm") {
		url = name
	}
	registration := &api.AgentServiceRegistration{
		ID:      id,
		Name:    name,
		Port:    port,
		Address: GetLocalIP().String(),
		Check: &api.AgentServiceCheck{
			HTTP:     fmt.Sprintf("http://%s:%d/health", url, port),
			Interval: "10s",
			Timeout:  "5s",
		},
	}
	if tags != nil && len(tags.Tags) > 0 {
		for _, tag := range tags.Tags {
			tg := strings.ReplaceAll(tag, "{{name}}", name)
			registration.Tags = append(registration.Tags, strings.ReplaceAll(tg, "{{port}}", fmt.Sprintf("%d", port)))
		}
	}
	c.logger.Info("Service ip", "id adr", url)
	err := c.api.Agent().ServiceRegister(registration)
	if err != nil {
		return err
	}
	slog.Info("Service enregistré avec protection Keycloak", "service", name, "id", id)
	// slog.Info("Service ip", "id adr", fmt.Sprintf("http://%s:%d/health", url, port))
	return nil
}

// Deregister removes the service from Consul
func (c *Client) Deregister(id string) {
	err := c.api.Agent().ServiceDeregister(id)
	if err != nil {
		c.logger.Error("Erreur lors de la désinscription", "id", id, "erreur", err)
	}
}

// GetKV retrieves a simple value from the KV Store
func (c *Client) GetKV(path string) (*Labels, error) {
	pair, _, err := c.api.KV().Get(path, nil)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, fmt.Errorf("clé introuvable : %s", path)
	}
	res := &Labels{}
	err = yaml.Unmarshal(pair.Value, res)
	if err != nil {
		return nil, fmt.Errorf("Error occured while processing tags : %s", err.Error())
	}
	return res, nil
}

// DiscoverRemotes finds services exposing a UI
func (c *Client) DiscoverRemotes(dm string) ([]RemoteModule, error) {
	services, _, err := c.api.Catalog().Services(nil)
	if err != nil {
		return nil, err
	}

	var remotes []RemoteModule
	for sName, tags := range services {
		isUI := false
		// slog.Info("Service encours de traitement", "service", sName)
		for _, t := range tags {
			if t == "knb-ui=true" {
				isUI = true
				break
			}
		}

		if isUI {
			cleanName := strings.TrimPrefix(sName, "knb-")
			remotes = append(remotes, RemoteModule{
				Name: sName,
				URL:  fmt.Sprintf("https://%s.%s/ui/remoteEntry.js", cleanName, dm),
			})
		}
	}
	return remotes, nil
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
