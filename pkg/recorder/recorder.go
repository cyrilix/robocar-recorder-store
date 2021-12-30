package recorder

import (
	"encoding/json"
	"fmt"
	"github.com/cyrilix/robocar-base/service"
	"github.com/cyrilix/robocar-protobuf/go/events"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
)

func New(client mqtt.Client, recordsDir, recordTopic string) (*Recorder, error) {
	err := os.MkdirAll(recordsDir, os.FileMode(0755))
	if err != nil {
		return nil, fmt.Errorf("unable to create %v directory: %v", recordsDir, err)
	}
	return &Recorder{
		client:      client,
		recordsDir:  recordsDir,
		recordTopic: recordTopic,
		cancel:      make(chan interface{}),
	}, nil

}

type Recorder struct {
	client      mqtt.Client
	recordsDir  string
	recordTopic string
	cancel      chan interface{}
}

var FileNameFormat = "record_%s.json"

func (r *Recorder) Start() error {
	err := service.RegisterCallback(r.client, r.recordTopic, r.onRecordMsg)
	if err != nil {
		return fmt.Errorf("unable to start rc-recorder part: %v", err)
	}
	<-r.cancel
	return nil
}

func (r *Recorder) Stop() {
	service.StopService("record", r.client, r.recordTopic)
	close(r.cancel)
}

func (r *Recorder) onRecordMsg(_ mqtt.Client, message mqtt.Message) {
	l := zap.S()
	var msg events.RecordMessage
	err := proto.Unmarshal(message.Payload(), &msg)
	if err != nil {
		zap.S().Errorf("unable to unmarshal protobuf %T: %v", msg, err)
		return
	}
	fmt.Printf("record %s: %s\r", msg.GetRecordSet(), msg.GetFrame().GetId().GetId())

	recordDir := fmt.Sprintf("%s/%s", r.recordsDir, msg.GetRecordSet())

	imgDir := fmt.Sprintf("%s/cam", recordDir)
	imgRef := fmt.Sprintf("cam/cam-image_array_%s.jpg", msg.GetFrame().GetId().GetId())
	imgName := fmt.Sprintf("%s/cam-image_array_%s.jpg", imgDir, msg.GetFrame().GetId().GetId())
	err = os.MkdirAll(imgDir, os.FileMode(0755))
	if err != nil {
		l.Errorf("unable to create %v directory: %v", imgDir, err)
		return
	}
	err = ioutil.WriteFile(imgName, msg.GetFrame().GetFrame(), os.FileMode(0755))
	if err != nil {
		l.Errorf("unable to write img file %v: %v", imgName, err)
		return
	}

	jsonDir := fmt.Sprintf("%s/", recordDir)
	recordName := fmt.Sprintf("%s/%s", jsonDir, fmt.Sprintf(FileNameFormat, msg.GetFrame().GetId().GetId()))
	err = os.MkdirAll(jsonDir, os.FileMode(0755))
	if err != nil {
		l.Errorf("unable to create %v directory: %v", jsonDir, err)
		return
	}
	record := Record{
		UserAngle:     msg.GetSteering().GetSteering(),
		CamImageArray: imgRef,
	}
	jsonBytes, err := json.Marshal(&record)
	if err != nil {
		l.Errorf("unable to marshal json content: %v", err)
		return
	}
	err = ioutil.WriteFile(recordName, jsonBytes, 0755)
	if err != nil {
		l.Errorf("unable to write json file %v: %v", recordName, err)
	}

}

type Record struct {
	UserAngle     float32 `json:"user/angle,"`
	CamImageArray string  `json:"cam/image_array,"`
}
