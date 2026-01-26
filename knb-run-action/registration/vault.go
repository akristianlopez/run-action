package registration

import (
	"errors"
	"log"

	"github.com/hashicorp/vault/api"
)

func Read(addr, token, path /*, role, pass*/ string) (string, error) {
	config := api.DefaultConfig()
	config.Address = addr
	client, err := api.NewClient(config)
	if err != nil {
		return "", err
	}
	client.SetToken(token)
	secret, err := client.Logical().Read(path)
	if err != nil {
		// Gérer l'erreur
		return "", err
	}
	if secret == nil {
		log.Fatal("Connection to the vault server is impossible")
	}
	// Accéder à la valeur
	// value := secret.Data["data"].(map[string]interface{})
	value := secret.Data ///[key].(map[string]interface{})
	result := ""
	if v, ok := value["password"].(string); ok {
		result = v
	} else {
		return "", errors.New("Invalid secret assertion")
	}
	return result, nil
}
func Write(addr, token, path, key /* role, pass,*/, value string) (bool, error) {
	config := api.DefaultConfig()
	config.Address = addr
	client, err := api.NewClient(config)
	if err != nil {
		return false, err
	}
	client.SetToken(token)

	// // Exemple simplifié, la configuration d'AppRole est plus complexe
	// loginData := map[string]interface{}{
	// 	"role_id":   role,n
	// 	"secret_id": pass,
	// }
	// resp, err := client.Logical().Write("auth/approle/login", loginData)
	// client.SetToken(resp.Auth.ClientToken)

	data := make(map[string]interface{})
	data[key] = value
	_, err = client.Logical().Write(path, data)
	if err != nil {
		// Gérer l'erreur
		return false, err
	}
	return true, nil
}
