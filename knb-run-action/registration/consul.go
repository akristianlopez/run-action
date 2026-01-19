package registration

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/consul/api"
)

var token string

func Register(port int, addr, kind string) (string, error) {
	config := api.DefaultConfig()
	consulClient, err := api.NewClient(config)
	if err != nil {
		log.Fatalf("Error when trying to create a consul client: %v", err)
	}
	serviceID := fmt.Sprintf("wosa-%d", port)
	serviceName := "wosa-run"
	if !strings.EqualFold(kind, "action") {
		serviceName = "wosa-query"
	}
	servicePort := port    //4000
	serviceAddress := addr //"localhost"
	registration := &api.AgentServiceRegistration{ID: serviceID, Name: serviceName,
		Port:    servicePort,
		Address: serviceAddress,
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("https://%s:%d/health", serviceAddress, servicePort),
			Interval:                       "10s", //Verifier toutes les 10 secondes
			Timeout:                        "30s", //Timeout 30 secondes
			DeregisterCriticalServiceAfter: "60s", //Se desenregistrer apres 60 secondes si le service est critique
		},
	}
	err = consulClient.Agent().ServiceRegister(registration)
	if err != nil {
		log.Fatalf("Error during the service registration: %v", err)
	}
	return serviceName, nil
}
