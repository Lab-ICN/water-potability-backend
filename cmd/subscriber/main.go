package main

import (
	"context"
	"encoding/json"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/joho/godotenv/autoload"
	"github.com/lab-icn/water-potability-sensor-service/internal/config"
	"github.com/lab-icn/water-potability-sensor-service/internal/grpc"
	"github.com/lab-icn/water-potability-sensor-service/internal/influxdb"
	mqttDelivery "github.com/lab-icn/water-potability-sensor-service/internal/interface/mqtt"
	pb "github.com/lab-icn/water-potability-sensor-service/internal/interface/rpc"
	_mqtt "github.com/lab-icn/water-potability-sensor-service/internal/mqtt"
	"github.com/lab-icn/water-potability-sensor-service/internal/repository"
	"github.com/lab-icn/water-potability-sensor-service/internal/service"
	"github.com/rs/zerolog"
)

func main() {
	path := os.Getenv("CONFIG_FILEPATH")
	content, err := os.ReadFile(path)
	if err != nil {
		stdlog.Fatalf("config: opening config file at %s: %v\n", path, err)
	}
	cfg := new(config.Config)
	if err := json.Unmarshal(content, cfg); err != nil {
		stdlog.Fatalf("config: parsing config file content: %v\n", err)
	}
	ctx := context.Background()

	log := zerolog.
		New(os.Stderr).
		With().
		Timestamp().
		Logger().
		Level(zerolog.DebugLevel)
	log.Info().Msg("app: starting")

	grpcClient, err := grpc.NewClient(cfg)
	if err != nil {
		stdlog.Fatalf("grpc: failed to establish connection: %v\n", err)
	}
	defer grpcClient.Close()

	log.Info().Msg("grpc: connection established")

	influxdb, err := influxdb.NewClient(ctx, &cfg.InfluxDB)
	if err != nil {
		stdlog.Fatalf("influxdb: failed to establish connection: %v\n", err)
	}
	defer influxdb.Close()

	log.Info().Msg("influxdb: connection established")

	wpClient := pb.NewWaterPotabilityServiceClient(grpcClient)
	wpRepository := repository.NewWaterPotabilityRepository(influxdb, &cfg.InfluxDB)
	wpService := service.NewWaterPotabilityService(wpRepository, wpClient, log)
	subscriber := mqttDelivery.NewMqttSubscriber(wpService, cfg, &log)

	mqtt, err := _mqtt.Listen(subscriber, cfg, &log)
	if err != nil {
		stdlog.Fatalf("mqtt: failed to establish connection: %v\n", err)
	}
	defer mqtt.Disconnect(250)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGILL, syscall.SIGTERM)
	done := make(chan struct{}, 1)
	go func() {
		<-sig
		stdlog.Println("app: shutting down...")
		done <- struct{}{}
	}()
	<-done
	stdlog.Println("app: exiting...")
}
