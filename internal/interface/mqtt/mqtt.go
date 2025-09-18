package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/lab-icn/water-potability-sensor-service/internal/aes256"
	"github.com/lab-icn/water-potability-sensor-service/internal/config"
	"github.com/lab-icn/water-potability-sensor-service/internal/domain"
	"github.com/lab-icn/water-potability-sensor-service/internal/service"
	"github.com/rs/zerolog"
)

type subscriber struct {
	service service.WaterPotabilityServiceItf
	cfg     *config.AES
	log     *zerolog.Logger
}

type IMqttSubscriber interface {
	SensorSubscriber(client mqtt.Client, msg mqtt.Message)
}

func NewMqttSubscriber(
	service service.WaterPotabilityServiceItf,
	cfg *config.Config,
	log *zerolog.Logger,
) IMqttSubscriber {
	return &subscriber{service, &cfg.AES, log}
}

func (s *subscriber) SensorSubscriber(client mqtt.Client, msg mqtt.Message) {
	s.log.Debug().
		Msg(fmt.Sprintf(
			"mqtt: acting upon topic %s: %s",
			msg.Topic(),
			string(msg.Payload()),
		))

	jsonstr, err := aes256.Decrypt(
		string(msg.Payload()),
		[]byte(s.cfg.Key),
		[]byte(s.cfg.IV),
	)
	if err != nil {
		s.log.Err(err).
			Msg(fmt.Sprintf(
				"mqtt: failed to decrypting payload: %s",
				msg.Payload(),
			))
		return
	}
	s.log.Debug().Msg(fmt.Sprintf("mqtt: decrypted payload: %s", jsonstr))

	var potability domain.WaterPotability
	if err := json.Unmarshal([]byte(jsonstr), &potability); err != nil {
		s.log.Err(err).
			Msg(fmt.Sprintf("mqtt: failed to parse json string: %s", jsonstr))
		return
	}

	potability.Node = msg.Topic()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := s.service.PredictWaterPotability(ctx, potability); err != nil {
		s.log.Err(err).Msg("mqtt: failed to inference metrics")
		return
	}
}
