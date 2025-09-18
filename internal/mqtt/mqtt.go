package mqtt

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/lab-icn/water-potability-sensor-service/internal/config"
	mqttDelivery "github.com/lab-icn/water-potability-sensor-service/internal/interface/mqtt"
	"github.com/rs/zerolog"
)

func Listen(
	subscriber mqttDelivery.IMqttSubscriber,
	cfg *config.Config,
	log *zerolog.Logger,
) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(fmt.Sprintf(
			"%s://%s:%d",
			cfg.MQTT.Protocol,
			cfg.MQTT.Host,
			cfg.MQTT.Port,
		)).
		SetKeepAlive(15 * time.Second).
		SetPingTimeout(10 * time.Second).
		SetAutoReconnect(true).
		SetDefaultPublishHandler(func(_ mqtt.Client, _ mqtt.Message) {}).
		SetConnectionLostHandler(func(_ mqtt.Client, _ error) {
			log.Warn().Msg("mqtt: connection lost")
		}).
		SetReconnectingHandler(func(_ mqtt.Client, _ *mqtt.ClientOptions) {
			log.Info().Msg("mqtt: reconnecting")
		}).
		SetOnConnectHandler(func(client mqtt.Client) {
			log.Info().Msg("mqtt: connection established")

			errChan := make(chan error)

			for _, topic := range cfg.MQTT.SensorTopics {
				go func(topic string) {
					token := client.Subscribe(topic, cfg.MQTT.QOS, subscriber.SensorSubscriber)
					<-token.Done()
					if err := token.Error(); err != nil {
						errChan <- err
					}
				}(topic)
			}

			if err := <-errChan; err != nil {
				log.Error().Err(err).Msg("mqtt: topic subscription failed")
			}
		})

	client := mqtt.NewClient(opts)

	token := client.Connect()
	<-token.Done()
	if err := token.Error(); err != nil {
		return nil, err
	}

	return client, nil
}
