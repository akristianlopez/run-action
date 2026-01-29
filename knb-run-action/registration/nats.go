package registration

import (
	"log"

	"github.com/akristianlopez/run-action/knb-run-action/webapi"
	"github.com/nats-io/nats.go"
)

func Subscrib(url, topic string) error {
	nc, err := nats.Connect(url)
	if err != nil {
		log.Printf("The subscrition is not possible %v", err)
		return err
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		log.Printf("The subscrition is not possible %v", err)
		return err
	}
	sub, err := js.Subscribe(topic, func(m *nats.Msg) {
		if webapi.HandleBrokerMessage(m.Data) {
			m.Ack()
		}
	}, nats.Durable(webapi.ConfigClient.Params["service_name"].(string)), nats.ManualAck())
	if err != nil {
		log.Println(err)
	}
	defer sub.Unsubscribe()
	log.Println("En attente de messages...")
	select {} //gader le programme actif
}
