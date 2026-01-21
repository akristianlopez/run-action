package registration

import (
	"errors"
	"log"
	"strconv"

	"github.com/akristianlopez/run-action/knb-run-action/webapi"
	"github.com/hashicorp/vault/api"
)

func Read(addr, token, path, key /*, role, pass*/ string) (*webapi.Db_access_params, error) {
	config := api.DefaultConfig()
	config.Address = addr
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	client.SetToken(token)

	// // Exemple simplifié, la configuration d'AppRole est plus complexe
	// loginData := map[string]interface{}{
	// 	"role_id":   role,
	// 	"secret_id": pass,
	// }
	// resp, err := client.Logical().Write("auth/approle/login", loginData)
	// client.SetToken(resp.Auth.ClientToken)

	secret, err := client.Logical().Read(path)
	if err != nil {
		// Gérer l'erreur
		return nil, err
	}
	if secret == nil {
		log.Fatal("Connection to the vault server is impossible")
	}
	// Accéder à la valeur
	// value := secret.Data["data"].(map[string]interface{})
	value := secret.Data ///[key].(map[string]interface{})
	result := &webapi.Db_access_params{}
	if v, ok := value["userid"].(string); ok {
		result.Userid = v
	} else {
		return nil, errors.New("parameter 'userid' is not defined")
	}
	if v, ok := value["password"].(string); ok {
		result.Password = v
	} else {
		return nil, errors.New("parameter 'password' is not defined")
	}
	if v, ok := value["port"].(string); ok {
		i, er := strconv.Atoi(v)
		if er != nil {
			return nil, er
		}
		result.Port = int64(i)
	} else {
		return nil, errors.New("parameter 'port' is not defined")
	}
	if v, ok := value["address"].(string); ok {
		result.Address = v
	} else {
		return nil, errors.New("parameter 'address' is not defined")
	}
	if v, ok := value["db"].(string); ok {
		result.Name = v
	} else {
		return nil, errors.New("parameter 'database name' is not defined")
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
