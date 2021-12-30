package main

import (
	"flag"
	"github.com/cyrilix/robocar-base/cli"
	"github.com/cyrilix/robocar-recorder-store/pkg/recorder"
	"go.uber.org/zap"
	"log"
	"os"
)

const (
	DefaultClientId = "robocar-rc-recorder"
)

func main() {
	var mqttBroker, username, password, clientId string
	var recordTopic string
	var recordsPath string

	mqttQos := cli.InitIntFlag("MQTT_QOS", 0)
	_, mqttRetain := os.LookupEnv("MQTT_RETAIN")

	cli.InitMqttFlags(DefaultClientId, &mqttBroker, &username, &password, &clientId, &mqttQos, &mqttRetain)
	flag.StringVar(&recordTopic, "mqtt-topic-records", os.Getenv("MQTT_TOPIC_RECORDS"), "Mqtt topic that contains record data for training, use MQTT_TOPIC_RECORDS if args not set")
	flag.StringVar(&recordsPath, "record-path", os.Getenv("RECORD_PATH"), "Path where to write records files, use RECORD_PATH if args not set")
	logLevel := zap.LevelFlag("log", zap.InfoLevel, "log level")
	flag.Parse()

	if len(os.Args) <= 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(*logLevel)
	lgr, err := config.Build()
	if err != nil {
		log.Fatalf("unable to init logger: %v", err)
	}
	defer func() {
		if err := lgr.Sync(); err != nil {
			log.Printf("unable to Sync logger: %v\n", err)
		}
	}()
	zap.ReplaceGlobals(lgr)

	client, err := cli.Connect(mqttBroker, username, password, clientId)
	if err != nil {
		zap.S().Fatalf("unable to connect to mqtt bus: %v", err)
	}
	defer client.Disconnect(50)

	r, err := recorder.New(client, recordsPath, recordTopic)
	if err != nil {
		zap.S().Fatalf("unable to init rc-recorder: %v", err)
	}
	defer r.Stop()

	cli.HandleExit(r)

	err = r.Start()
	if err != nil {
		zap.S().Fatalf("unable to start service: %v", err)
	}
}
