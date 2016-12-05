package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"log"
	"os/exec"
	"time"
)

var mqttPort = flag.String("mqttPort", "1883", "mqttPort of broker")
var qos = flag.Int("qos", 0, "The QoS to subscribe to messages at")
var clientid = flag.String("clientid", "NodeHopeMqttDaemon", "A clientid for the connection")
var user = flag.String("user", "wifi", "username")
var pass = flag.String("pass", "68008232", "password")
var topic = flag.String("topic", "mqttdaemon", "topic")
var mqttBin = flag.String("mqttBin", "/usr/emqttd/bin/emqttd", "mqttBin like /usr/emqttd/bin/emqttd start")
var mqttBinParamter = flag.String("mqttBinParamter", "console", "mqttBin mqttBinParamter")

type mqttDaemon struct{
	aliveCount int64
	client mqtt.Client;
}


func (this *mqttDaemon)init(){
	this.aliveCount = 0
	this.conn()
}

/*
从mqtt接受消息，并把消息转发到浏览器
*/
func (this *mqttDaemon)onMessageReceivedFromMqtt(client mqtt.Client, message mqtt.Message) {
	//fmt.Printf("Received message on topic from mqtt: %s\n内容为: %s\n", message.Topic(), message.Payload())
	this.aliveCount++
	fmt.Println("aliveCount: " , this.aliveCount)
}

func (this *mqttDaemon)OnConnectHandler(client mqtt.Client){
	if token := client.Subscribe(*topic, byte(*qos), this.onMessageReceivedFromMqtt); token.Wait() && token.Error() != nil {
		fmt.Println("订阅topic: " + *topic + "失败。" + token.Error().Error())
	}else{
		fmt.Println("订阅topic: " + *topic + "成功")
	}

}

func (this *mqttDaemon)ConnectionLostHandler(client mqtt.Client, err error){
	fmt.Println("连接断开" + err.Error())
	//启动mqtt程序
	dateCmd := exec.Command(*mqttBin, *mqttBinParamter)
	dateOut, err := dateCmd.Output()
	if err != nil {
		fmt.Println("启动mqtt：" + *mqttBin + " " + *mqttBinParamter + "失败！error:" + err.Error())
	}else{
		fmt.Println("启动mqtt：成功， output：" + string(dateOut))
	}
}

func(this *mqttDaemon)conn(){
	connOpts := &mqtt.ClientOptions{
		ClientID:             *clientid,
		CleanSession:         true,
		Username:             *user,
		Password:             *pass,
		ConnectTimeout:       3 * time.Second,
		AutoReconnect:        true,
		MaxReconnectInterval: 3 * time.Second,
		KeepAlive:            30 * time.Second,
		OnConnect:              this.OnConnectHandler,
		OnConnectionLost:       this.ConnectionLostHandler,
		TLSConfig:            tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert},
	}
	connOpts.AddBroker("tcp://101.201.37.95:" + *mqttPort)
	this.client = mqtt.NewClient(connOpts)
	if token := this.client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		//panic(token.Error())

	} else {
		fmt.Printf("Connected to %s\n", *mqttPort)
	}
}

func (this *mqttDaemon)updateAlive(){
	this.client.Publish(*topic, 1, false, "alive")
}


func main() {
	flag.Parse()
	log.SetFlags(0)
	var damon mqttDaemon
	damon.init()
	//死循环更新数据
	for {
		time.Sleep(1000 * time.Millisecond)
		damon.updateAlive()
	}
}
