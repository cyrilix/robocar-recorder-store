package recorder

import (
	"encoding/json"
	"fmt"
	"github.com/cyrilix/robocar-base/testtools"
	"github.com/cyrilix/robocar-protobuf/go/events"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/protobuf/ptypes/timestamp"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

func TestRecorder_onRecordMsg(t *testing.T) {
	type fields struct {
		recordsDir string
		recordSet  string
	}
	type args struct {
		message mqtt.Message
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		wantJsonFileName string
		wantRecord       Record
	}{
		{
			name: "default",
			fields: fields{
				recordsDir: t.TempDir(),
				recordSet:  "default",
			},
			args: args{
				message: generateMessage("1", "default", -0.5, 0.6, events.DriveMode_PILOT),
			},
			wantJsonFileName: "record_1.json",
			wantRecord: Record{
				UserAngle:      -0.5,
				CamImageArray:  "cam/cam-image_array_1.jpg",
				AutopilotAngle: 0.6,
				DriveMode:      events.DriveMode_PILOT.String(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Recorder{
				recordsDir: tt.fields.recordsDir,
			}
			r.onRecordMsg(nil, tt.args.message)
			fis, err := os.ReadDir(tt.fields.recordsDir)
			if err != nil {
				t.Errorf("unable to list files: %v", err)
				return
			}
			if len(fis) != 1 {
				t.Errorf("bad number of entry into %v: %v, want %v", tt.fields.recordsDir, len(fis), 1)
			}
			if !fis[0].IsDir() {
				t.Errorf("target record is not a directory")
			}
			if fis[0].Name() != tt.name {
				t.Errorf("bad directory name '%v', want '%v'", fis[0].Name(), tt.fields.recordSet)
			}
			records, err := os.ReadDir(path.Join(tt.fields.recordsDir, fis[0].Name()))
			if err != nil {
				t.Errorf("unable to list record files")
				return
			}
			if len(records) != 2 {
				t.Errorf("records have %d entry, want %d", len(records), 2)
				return
			}

			var record Record
			for _, r := range records {
				if r.IsDir() {
					if r.Name() != "cam" {
						t.Errorf("bad name for cam records '%v', want %v", r.Name(), "cam")
					}
					continue
				}

				if r.Name() != tt.wantJsonFileName {
					t.Errorf("bad json filename '%v', want '%v'", r.Name(), tt.wantJsonFileName)
				}
				jsonContent, err := ioutil.ReadFile(path.Join(tt.fields.recordsDir, tt.fields.recordSet, r.Name()))
				if err != nil {
					t.Errorf("unable to read json record: %v", err)
				}
				err = json.Unmarshal(jsonContent, &record)
				if err != nil {
					t.Errorf("unable to unmarshal record: %v", err)
					return
				}
			}

			if record != tt.wantRecord {
				t.Errorf("bad json record '%v', want '%v'", record, tt.wantRecord)
			}

			img, err := ioutil.ReadFile(path.Join(tt.fields.recordsDir, tt.fields.recordSet, record.CamImageArray))
			if err != nil {
				t.Errorf("unable to read image: %v", err)
				return
			}
			if string(img) != "frame content" {
				t.Errorf("bad image content")
			}
		})
	}
}

func generateMessage(id string, recordSet string, userAngle float32, autopilotAngle float32,
	driveMode events.DriveMode) mqtt.Message {
	now := time.Now()
	msg := events.RecordMessage{
		Frame: &events.FrameMessage{
			Id: &events.FrameRef{
				Name: fmt.Sprintf("framie-%s", id),
				Id:   id,
				CreatedAt: &timestamp.Timestamp{
					Seconds: now.Unix(),
					Nanos:   int32(now.Nanosecond()),
				},
			},
			Frame: []byte("frame content"),
		},
		Steering: &events.SteeringMessage{
			Steering:   userAngle,
			Confidence: 1.0,
			FrameRef: &events.FrameRef{
				Name: fmt.Sprintf("framie-%s", id),
				Id:   id,
				CreatedAt: &timestamp.Timestamp{
					Seconds: now.Unix(),
					Nanos:   int32(now.Nanosecond()),
				},
			},
		},
		DriveMode: &events.DriveModeMessage{
			DriveMode: driveMode,
		},
		AutopilotSteering: &events.SteeringMessage{
			Steering:   autopilotAngle,
			Confidence: 0.8,
			FrameRef: &events.FrameRef{
				Name: fmt.Sprintf("framie-%s", id),
				Id:   id,
				CreatedAt: &timestamp.Timestamp{
					Seconds: now.Unix(),
					Nanos:   int32(now.Nanosecond()),
				},
			},
		},
		RecordSet: recordSet,
	}

	return testtools.NewFakeMessageFromProtobuf("topic", &msg)
}
