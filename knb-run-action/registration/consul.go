package registration

import (
	"fmt"
	"log"

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
