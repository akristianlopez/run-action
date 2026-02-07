package registration

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/akristianlopez/run-action/knb-run-action/webapi"
	"github.com/streadway/amqp"
)

func connectBroker(url string) *amqp.Connection {
	if url == "" {
		slog.Error("No available brokers", "url", url)
		return nil
	}
	conn, err := amqp.Dial(url)
	if err == nil {
		slog.Error("Impossible to establish the connection to the broker", "url", url)
		return nil
	}
	return conn
}

func RabbitMQSubscrib(url, topic string) error {
	conn := connectBroker(url)
	if conn == nil {
		slog.Error("❌ Error while trying to establish the connection to the broker. \r\n The service was stopped.")
		os.Exit(1)
	}
	defer conn.Close()
	// 1. Création d'un canal
	ch, err := conn.Channel()
	if err != nil {
		slog.Error("❌ Error while trying to open a chanel", "error", err)
		os.Exit(1)
	}
	defer ch.Close()

	// 2. Déclaration de la queue (doit être identique à celle du producteur)
	q, err := ch.QueueDeclare(
		topic, // nom de la queue
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)

	// 3. Réception des messages
	msgs, err := ch.Consume(
		q.Name,
		"",    // consumer
		false, // auto-ack : mis à false pour ne pas perdre de messages en cas de crash
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)

	// 4. Boucle de traitement
	go func() {
		for d := range msgs {
			// slog.Warn("📥 Message received ", "error", d.Body)

			// --- VALIDATION SÉCURITÉ ---
			// On récupère le token passé dans les headers du message
			token, ok := d.Headers["x-auth-token"].(string)
			if !ok || !webapi.ValidateToken(token, webapi.ConfigClient.Params["jwt_key"].(string)) {
				slog.Warn("⚠️ Rejected message : Invalide Token")
				d.Nack(false, false) // Rejeter sans remettre en file
				continue
			}

			// --- TRAITEMENT MÉTIER ---
			// err := processTask(d.Body)
			if webapi.HandleBrokerMessage(url, fmt.Sprintf("%s.*", d.MessageId), d.RoutingKey, token, d.Body) {
				d.Ack(false)
			}

			if err != nil {
				slog.Error("❌ Erreur traitement", "error", err)
				d.Nack(false, true) // Remettre en file pour réessayer plus tard
			} else {
				d.Ack(false) // Accuser réception : le message est supprimé de RabbitMQ
			}
		}
	}()

	slog.Info("👤 Ready to treat the message...")
	return nil
}

func RabbitMQPublish(url, subj, message, jwtToken string) (bool, error) {
	conn := connectBroker(url)
	if conn == nil {
		slog.Error("❌ Error while trying to establish the connection to the broker. \r\n The service was stopped.")
		os.Exit(1)
	}
	defer conn.Close()
	// 1. Ouverture d'un canal unique pour l'envoi
	ch, err := conn.Channel()
	if err != nil {
		return false, fmt.Errorf("erreur canal: %v", err)
	}
	defer ch.Close()

	// 2. Déclaration de l'Exchange (le point d'entrée des messages)
	// Type "direct" est idéal pour router vers une queue spécifique
	err = ch.ExchangeDeclare(
		"knb_exchange", // nom
		"direct",       // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		return false, err
	}

	// 3. Préparation du message avec Headers
	msg := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent, // Le message survit au redémarrage du broker
		Headers: amqp.Table{
			"x-auth-token": jwtToken, // Injection du JWT pour la sécurité
		},
		Body: []byte(message),
	}

	// 4. Publication
	err = ch.Publish(
		"knb_exchange", // exchange
		subj,           // routing key (ex: "run.task")
		false,          // mandatory
		false,          // immediate
		msg,
	)

	if err == nil {
		slog.Info("📤 Message publié avec succès", "key", subj)
		return true, nil
	}
	return false, err
}
