package registration

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/akristianlopez/run-action/knb-run-action/webapi"
	"github.com/segmentio/kafka-go"
)

func KafkaPublish(url, subj, message, token string) (bool, error) {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(url),
		Topic:    subj,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	// Exemple d'envoi d'un message
	err := writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte("user-123"), // Permet de garantir l'ordre par utilisateur
			Value: []byte(message),
			Headers: []kafka.Header{
				{Key: "x-auth-token", Value: []byte(token)},
			},
		},
	)
	if err != nil {
		slog.Error("❌ Erreur de publication Kafka", "error", err)
		return false, err
	}
	return true, nil
}

func KafkaSubscrib(brokerAddr string, topic string) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddr},
		GroupID:  "knb-run-group", // Toutes les répliques avec le même ID se partagent le travail
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	defer reader.Close()

	slog.Info("👂 Consommateur Kafka prêt...")

	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			slog.Error("❌ Erreur lecture Kafka", "error", err)
			return err
		}

		// --- EXTRACTION ET VALIDATION DU TOKEN ---
		var token string
		for _, h := range m.Headers {
			if h.Key == "x-auth-token" {
				token = string(h.Value)
				break
			}
		}

		if !webapi.ValidateToken(token, webapi.ConfigClient.Params["jwt_key"].(string)) {
			slog.Warn("⚠️ Message ignoré : Token JWT invalide")
			continue
		}

		// --- TRAITEMENT MÉTIER ---
		slog.Info(fmt.Sprintf("📥 Message reçu à l'offset %d: %s = %s", m.Offset, string(m.Key), string(m.Value)))
		if !webapi.HandleBrokerMessage(brokerAddr, m.Topic, "", token, m.Value) {
			slog.Error(fmt.Sprintf("Error while trying to treat the message '%s'", m.Topic), "value", m.Value)
		}
		// Le commit (Ack) est automatique avec ReadMessage par défaut
	}
}
