package registration

import (
	"fmt"
	"log"
	"strings"

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
	//webapi.Emit=Emit
	sub, err := js.Subscribe(topic, func(m *nats.Msg) {
		t := strings.Split(m.Subject, ".")
		msg := t[0]
		if len(t) > 1 {
			msg = strings.Join(t[1:], ".")
		}
		if webapi.HandleBrokerMessage(nc.ConnectedUrl(), fmt.Sprintf("%s.*", t[0]), msg, m.Data) {
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

func Emit(url string, subj string, message string) (bool, error) {
	con, err := nats.Connect(url)
	if err != nil {
		log.Printf("The subscrition is not possible %v", err)
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
