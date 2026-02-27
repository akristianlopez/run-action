package discoveries

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/akristianlopez/run-action/knb-run-action/webapi"
	"github.com/hashicorp/consul/api"
)

// var token string

func Register(port uint64, name, addr, laddr, kind string, tags []string) (string, error) {
	config := api.DefaultConfig()
	config.Address = addr
	consulClient, err := api.NewClient(config)
	if err != nil {
		slog.Error("Error when trying to create a consul client", "error", err)
		os.Exit(1)
	}
	hn, _ := os.Hostname()
	serviceID := hn //fmt.Sprintf("%s:%d/%s", name, port, kind)
	serviceName := name
	// if !strings.EqualFold(kind, "action") {
	// 	serviceName = "wosa-query"
	// }
	servicePort := port     //4000
	serviceAddress := laddr //"localhost"
	registration := &api.AgentServiceRegistration{ID: serviceID, Name: serviceName,
		Port:    int(servicePort),
		Address: serviceAddress,
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d/action/ping", serviceAddress, servicePort),
			Interval:                       fmt.Sprintf("%ds", webapi.ConfigClient.Params["check_health_interval"].(int)), //Verifier toutes les 10 secondes
			Timeout:                        fmt.Sprintf("%ds", webapi.ConfigClient.Params["timeout"].(int)),               //Timeout 30 secondes
			DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", webapi.ConfigClient.Params["deregistry_delay_time"].(int)), //Se desenregistrer apres 60 secondes si le service est critique
		},
	}
	if len(tags) > 0 {
		for _, tag := range tags {
			tg := strings.ReplaceAll(tag, "{{name}}", name)
			registration.Tags = append(registration.Tags, strings.ReplaceAll(tg, "{{port}}", fmt.Sprintf("%d", port)))
		}
	}

	err = consulClient.Agent().ServiceRegister(registration)
	if err != nil {
		slog.Error("Error during the service registration", "error", err)
		os.Exit(1)
	}
	return serviceName, nil
}
func RegisterEx(port string, app_name, c_url, host_name string) error {
	config := api.DefaultConfig()
	config.Address = c_url
	consulClient, err := api.NewClient(config)
	if err != nil {
		slog.Error("Error when trying to create a consul client", "error", err)
		os.Exit(1)
	}
	serviceID := host_name
	serviceName := app_name
	servicePort := port         //4000
	serviceAddress := host_name //"localhost"
	p := 0
	_, err = fmt.Sscanf(port, "%d", &p)
	if err != nil {
		slog.Error("Error when trying to read the server port", "error", err)
		os.Exit(1)
	}

	registration := &api.AgentServiceRegistration{ID: serviceID, Name: serviceName,
		Port:    p,
		Address: serviceAddress,
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%s/action/ping", serviceAddress, servicePort),
			Interval:                       fmt.Sprintf("%ds", webapi.ConfigClient.Params["check_health_interval"].(int)), //Verifier toutes les 10 secondes
			Timeout:                        fmt.Sprintf("%ds", webapi.ConfigClient.Params["timeout"].(int)),               //Timeout 30 secondes
			DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", webapi.ConfigClient.Params["deregistry_delay_time"].(int)), //Se desenregistrer apres 60 secondes si le service est critique
		},
	}
	err = consulClient.Agent().ServiceRegister(registration)
	if err != nil {
		slog.Error("Error during the service registration", "error", err)
		os.Exit(1)
	}
	return nil
}
func Deregister(serID, c_url string) error {
	config := api.DefaultConfig()
	config.Address = c_url
	consulClient, err := api.NewClient(config)
	if err != nil {
		slog.Error("Error when trying to create a consul client", "error", err)
		os.Exit(1)
	}

	err = consulClient.Agent().ServiceDeregister(serID)
	if err != nil {
		slog.Error(fmt.Sprintf("Error occured while trying to deregister the service '%s' from service discovery: %v", serID, err))
		os.Exit(1)
	}
	return nil
}
func ExistingService() ([]*api.ServiceEntry, error) {
	// Get a new client
	config := api.DefaultConfig()
	config.Address = webapi.ConfigClient.Params["discovery_service_address"].(string) // e.g., "localhost:8500"
	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("error creating consul client: %w", err)
	}
	serviceName := webapi.ConfigClient.Params["service_name"].(string)
	// Query the health catalog for the service
	// "passingOnly" set to true ensures we only see instances that are passing their health checks.
	passingOnly := true

	serviceEntries, _, err := client.Health().Service(serviceName, "", passingOnly, nil)
	if err != nil {
		// This might return an error if the service is completely unknown to Consul
		// or if there's a network issue.
		return nil, fmt.Errorf("error querying service health: %w", err)
	}

	// If the list of service entries is not empty, the service exists and has healthy instances.
	if len(serviceEntries) > 0 {
		return serviceEntries, nil
	}

	// If the list is empty, the service name might exist but have no healthy instances,
	// or it might not exist at all (depending on specific Consul configuration/behavior).
	// You can refine this by querying the full catalog if needed.
	return nil, errors.New("Service not found")
}
func IsServiceExists(entries []*api.ServiceEntry, name string) *api.ServiceEntry {
	if entries == nil {
		return nil
	}
	for _, entry := range entries {
		if strings.EqualFold(entry.Service.Service, name) {
			return entry
		}
	}
	return nil
}

// Get a new client
