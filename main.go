package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var gwHost string
var gwPort int64
var gwUsername string
var gwPassword string

var mqttHost string
var mqttPort int64
var mqttUsername string
var mqttPassword string

var mqttClient mqtt.Client

func main() {
	initEnv()
	initMqtt()
	mqttSubscribe()

	for {
		logrus.Info("push state begin")
		err := pushState()
		if err != nil {
			logrus.WithError(err).Error("push state error")
		}
		time.Sleep(time.Second)
		logrus.Info("push state end")
	}
}

func initEnv() {
	gwHost = os.Getenv("GW_HOST")
	gwPort, _ = strconv.ParseInt(os.Getenv("GW_PORT"), 10, 64)
	gwUsername = os.Getenv("GW_USERNAME")
	gwPassword = os.Getenv("GW_PASSWORD")
	mqttHost = os.Getenv("MQTT_HOST")
	mqttPort, _ = strconv.ParseInt(os.Getenv("MQTT_PORT"), 10, 64)
	mqttUsername = os.Getenv("MQTT_USERNAME")
	mqttPassword = os.Getenv("MQTT_PASSWORD")
}

func initMqtt() {
	logrus.Info("mqtt connect begin")
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", mqttHost, mqttPort))
	opts.SetClientID(fmt.Sprintf("zhm-%s", uuid.New().String()))
	opts.SetUsername(mqttUsername)
	opts.SetPassword(mqttPassword)
	mqttClient = mqtt.NewClient(opts)
	connectToken := mqttClient.Connect()
	connectToken.Wait()
	err := connectToken.Error()
	if err != nil {
		logrus.WithError(err).Error("mqtt connect error")
		panic(err)
	}
	logrus.Info("mqtt connect done")
}

func mqttSubscribe() {
	buildUnit := func(topic string) *unit {
		topicSlice := strings.Split(topic, "/")
		oa, _ := strconv.ParseInt(topicSlice[1], 10, 64)
		ia, _ := strconv.ParseInt(topicSlice[2], 10, 64)
		return &unit{
			Oa:      oa,
			Ia:      ia,
			On:      -1,
			Mode:    -1,
			TempSet: "",
			Fan:     -1,
		}
	}
	mqttClient.Subscribe("zhonghong/+/+/power/set", 1, func(client mqtt.Client, message mqtt.Message) {
		logrus.Infof("message receive topic:%v payload:%v", message.Topic(), message.Payload())
		u := buildUnit(message.Topic())
		u.On = powerCommand(string(message.Payload()))
		err := setState(u)
		if err != nil {
			logrus.WithError(err).Errorf("set power state error u:%+v", u)
		}
		message.Ack()
	})
	mqttClient.Subscribe("zhonghong/+/+/mode/set", 1, func(client mqtt.Client, message mqtt.Message) {
		logrus.Infof("message receive topic:%v payload:%v", message.Topic(), message.Payload())
		u := buildUnit(message.Topic())
		u.Mode = modeCommand(string(message.Payload()))
		err := setState(u)
		if err != nil {
			logrus.WithError(err).Errorf("set mode state error u:%+v", u)
		}
		message.Ack()
	})
	mqttClient.Subscribe("zhonghong/+/+/temperature/set", 1, func(client mqtt.Client, message mqtt.Message) {
		logrus.Infof("message receive topic:%v payload:%v", message.Topic(), message.Payload())
		u := buildUnit(message.Topic())
		u.TempSet = string(message.Payload())
		err := setState(u)
		if err != nil {
			logrus.WithError(err).Errorf("set temperature state error u:%+v", u)
		}
		message.Ack()
	})
	mqttClient.Subscribe("zhonghong/+/+/fan/set", 1, func(client mqtt.Client, message mqtt.Message) {
		logrus.Infof("message receive topic:%v payload:%v", message.Topic(), message.Payload())
		u := buildUnit(message.Topic())
		u.Fan = fanModeCommand(string(message.Payload()))
		err := setState(u)
		if err != nil {
			logrus.WithError(err).Errorf("set fan state error u:%+v", u)
		}
		message.Ack()
	})
}

func setState(u *unit) error {
	list, err := listUnit()
	if err != nil {
		return errors.Wrap(err, "list unit error")
	}
	idx := -1
	for i, v := range list {
		if (v.Oa != u.Oa) || (v.Ia != u.Ia) {
			continue
		}
		idx = i
		break
	}
	if idx == -1 {
		return errors.New("oa and ia not found")
	}
	cu := list[idx]
	params := make(map[string]string)
	params["f"] = "18"
	params["p"] = "0"
	if u.On == -1 {
		params["on"] = fmt.Sprintf("%d", cu.On)
	} else {
		params["on"] = fmt.Sprintf("%d", u.On)
	}
	if u.Mode == -1 {
		params["mode"] = fmt.Sprintf("%d", cu.Mode)
	} else {
		params["mode"] = fmt.Sprintf("%d", u.Mode)
	}
	if u.TempSet == "" {
		params["tempSet"] = cu.TempSet
	} else {
		params["tempSet"] = u.TempSet
	}
	if u.Fan == -1 {
		params["fan"] = fmt.Sprintf("%d", cu.Fan)
	} else {
		params["fan"] = fmt.Sprintf("%d", u.Fan)
	}
	params["idx"] = fmt.Sprintf("%d", idx)
	logrus.Infof("set device state params:%+v", params)
	resp, err := gwRequest(params)
	if err != nil {
		return errors.Wrap(err, "gateway request error")
	}
	logrus.Infof("set device state resp:%v", resp)
	var respData struct {
		Err int64 `json:"err"`
	}
	err = json.Unmarshal([]byte(resp), &respData)
	if err != nil {
		return errors.Wrap(err, "data unmarshal error")
	}
	if respData.Err != 0 {
		return errors.New("gateway response error")
	}
	return nil
}

func gwRequest(params map[string]string) (string, error) {
	u := fmt.Sprintf("http://%s:%d/cgi-bin/api.html", gwHost, gwPort)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", errors.Wrap(err, "create request error")
	}
	queries := url.Values{}
	for k, v := range params {
		queries.Add(k, v)
	}
	req.URL.RawQuery = queries.Encode()
	authToken := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", gwUsername, gwPassword)))
	req.Header.Add("Authorization", fmt.Sprintf("Basic %v", authToken))
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", errors.Wrap(err, "invoke request error")
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "read response body error")
	}
	return string(respBody), nil
}

type unit struct {
	Oa      int64  `json:"oa"`
	Ia      int64  `json:"ia"`
	On      int64  `json:"on"`
	Mode    int64  `json:"mode"`
	TempSet string `json:"tempSet"`
	Fan     int64  `json:"fan"`
}

func pushState() error {
	list, err := listUnit()
	if err != nil {
		return errors.Wrap(err, "list unit error")
	}
	for _, u := range list {
		logrus.Infof("mqtt publish begin oa:%d ia:%d", u.Oa, u.Ia)
		err = mqttPublish(fmt.Sprintf("zhonghong/%d/%d/power/state", u.Oa, u.Ia), powerState(u.On))
		if err != nil {
			logrus.WithError(err).Errorf("publish power state error unit:%+v", u)
		}
		err = mqttPublish(fmt.Sprintf("zhonghong/%d/%d/mode/state", u.Oa, u.Ia), modeState(u.Mode))
		if err != nil {
			logrus.WithError(err).Errorf("publish mode state error unit:%+v", u)
		}
		err = mqttPublish(fmt.Sprintf("zhonghong/%d/%d/temperature/state", u.Oa, u.Ia), u.TempSet)
		if err != nil {
			logrus.WithError(err).Errorf("publish temperature state error unit:%+v", u)
		}
		err = mqttPublish(fmt.Sprintf("zhonghong/%d/%d/fan/state", u.Oa, u.Ia), fanModeState(u.Fan))
		if err != nil {
			logrus.WithError(err).Errorf("publish fan state error unit:%+v", u)
		}
		logrus.Infof("mqtt publish end oa:%d ia:%d", u.Oa, u.Ia)
	}
	return nil
}

func listUnit() ([]*unit, error) {
	params := make(map[string]string)
	params["f"] = "17"
	params["p"] = "0"
	resp, err := gwRequest(params)
	if err != nil {
		return nil, errors.Wrap(err, "gateway request error")
	}
	logrus.Infof("gateway data:%v", resp)
	var respData struct {
		Err  int64  `json:"err"`
		Unit []unit `json:"unit"`
	}
	err = json.Unmarshal([]byte(resp), &respData)
	if err != nil {
		return nil, errors.Wrap(err, "data unmarshal error")
	}
	if respData.Err != 0 {
		return nil, errors.New("gateway response error")
	}
	var list []*unit
	for k := range respData.Unit {
		list = append(list, &respData.Unit[k])
	}
	return list, nil
}

func mqttPublish(topic, payload string) error {
	publishToken := mqttClient.Publish(topic, 1, false, payload)
	publishToken.Wait()
	err := publishToken.Error()
	if err != nil {
		return errors.Wrap(err, "mqtt publish error")
	}
	return nil
}

func powerCommand(v string) int64 {
	if v == "on" {
		return 1
	} else if v == "off" {
		return 0
	} else {
		return 0
	}
}

func powerState(v int64) string {
	if v == 1 {
		return "on"
	} else if v == 0 {
		return "off"
	} else {
		return "off"
	}
}

func modeCommand(v string) int64 {
	if v == "cool" {
		return 1
	} else if v == "dry" {
		return 2
	} else if v == "fan_only" {
		return 4
	} else if v == "heat" {
		return 8
	} else {
		return 1
	}
}

func modeState(v int64) string {
	if v == 1 {
		return "cool"
	} else if v == 2 {
		return "dry"
	} else if v == 4 {
		return "fan_only"
	} else if v == 8 {
		return "heat"
	} else {
		return "cool"
	}
}

func fanModeCommand(v string) int64 {
	if v == "high" {
		return 1
	} else if v == "medium" {
		return 2
	} else if v == "low" {
		return 4
	} else {
		return 1
	}
}

func fanModeState(v int64) string {
	if v == 1 {
		return "high"
	} else if v == 2 {
		return "medium"
	} else if v == 4 {
		return "low"
	} else {
		return "high"
	}
}
