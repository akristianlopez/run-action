package registration

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/akristianlopez/run-action/knb-run-action/webapi"
	"github.com/nats-io/nats.go"
)

func NatsSubscrib(url, topic string) error {
	nc, err := nats.Connect(url)
	if err != nil {
		slog.Error("The subscrition is not possible", "error", err)
		return err
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		slog.Error("The subscrition is not possible", "error", err)
		return err
	}
	//webapi.Emit=Emit
	sub, err := js.Subscribe(topic, func(m *nats.Msg) {
		t := strings.Split(m.Subject, ".")
		msg := t[0]
		if len(t) > 1 {
			msg = strings.Join(t[1:], ".")
		}

		// 1. Extraire le token, le valider et l'injecter dans la fonction
		token, ok := m.Header["x-auth-token"]

		if !ok || !webapi.ValidateToken(strings.Join(token, ""),
			webapi.ConfigClient.Params["jwt_key"].(string)) {
			slog.Warn("⚠️ Rejected message : Invalide Token")
			m.Nak() // Rejeter sans remettre en file
		}
		if webapi.HandleBrokerMessage(nc.ConnectedUrl(), fmt.Sprintf("%s.*", t[0]), msg, strings.Join(token, ""), m.Data) {
			m.Ack()
		}
	}, nats.Durable(webapi.ConfigClient.Params["service_name"].(string)), nats.ManualAck())
	if err != nil {
		slog.Error("The subscrition is not possible", "error", err)
		return err
	}
	defer sub.Unsubscribe()
	slog.Info("En attente de messages...")
	select {} //gader le programme actif
}

func NatsPublish(url, subj, message, token string) (bool, error) {
	//Penser a integrer le jwt token de l'appelant
	con, err := nats.Connect(url)
	if err != nil {
		slog.Warn("The subscrition is not possible", "error", err)
		return false, err
	}
	defer con.Close()
	err = con.Publish(subj, []byte(message))
	if err != nil {
		return false, err
	}
	con.Flush()
	if err := con.LastError(); err != nil {
		return false, err
	}
	return true, nil
}
