package secrets

import (
	"log"

	infisical "github.com/infisical/go-sdk"
)

type SecretManager struct {
	client infisical.InfisicalClientInterface
}

func NewSecretManager(machineIdentityID, machineIdentitySecret string) *SecretManager {
	// Initialisation du client
	client := infisical.NewInfisicalClient(nil, infisical.Config{SiteUrl: "https://vault.wosa.local"})

	// Authentification via Machine Identity (idéal pour Docker Swarm)
	_, err := client.Auth().UniversalAuthLogin(machineIdentityID, machineIdentitySecret)
	// _, err := client.Auth().MachineIdentityLogin(machineIdentityID, machineIdentitySecret)
	if err != nil {
		log.Fatalf("Échec auth Infisical: %v", err)
	}

	return &SecretManager{client: client}
}

func (s *SecretManager) GetSecret(key, projectID, envSlug string) string {
	secret, err := s.client.Secrets().Retrieve(infisical.RetrieveSecretOptions{
		SecretKey:   key,
		ProjectID:   projectID,
		Environment: envSlug,
	})
	if err != nil {
		log.Printf("Erreur lors de la récupération du secret %s: %v", key, err)
		return ""
	}
	return secret.SecretValue
}
