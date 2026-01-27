package registration

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/akristianlopez/run-action/knb-run-action/webapi"
	"github.com/hashicorp/consul/api"
)

// var token string

func Register(port uint64, name, addr, laddr, kind string) (string, error) {
	config := api.DefaultConfig()
	config.Address = addr
	consulClient, err := api.NewClient(config)
	if err != nil {
		log.Fatalf("Error when trying to create a consul client: %v", err)
	}
	serviceID := fmt.Sprintf("%s:%d/%s", name, port, kind)
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
	err = consulClient.Agent().ServiceRegister(registration)
	if err != nil {
		log.Fatalf("Error during the service registration: %v", err)
	}
	return serviceName, nil
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
