package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"log"
	"os/exec"
	"time"
	"sync"
	"bufio"
	"io"
)

var mqttPort = flag.String("mqttPort", "61613", "mqttPort of broker")
var qos = flag.Int("qos", 0, "The QoS to subscribe to messages at")
var clientid = flag.String("clientid", "NodeHopeMqttDaemon", "A clientid for the connection")
var user = flag.String("user", "wifi", "username")
var pass = flag.String("pass", "68008232", "password")
var topic = flag.String("topic", "mqttdaemon", "topic")
var emqttShellPath = flag.String("emqttShellPath", "/usr/emqttd/bin/emqttd", "emqtt shell")

type mqttDaemon struct{
	aliveCount int64
	client mqtt.Client;

	aliveCountMux sync.Mutex
}

func(this *mqttDaemon)timerCheck(){
	for {
		time.Sleep(3000 * time.Millisecond)
		this.updateAlive();
	}
}

func (this *mqttDaemon)init(){
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
	connOpts.AddBroker("tcp://127.0.0.1:" + *mqttPort)
	this.client = mqtt.NewClient(connOpts)
	this.conn()
	go this.timerCheck()
}

/*
从mqtt接受消息，并把消息转发到浏览器
*/
func (this *mqttDaemon)onMessageReceivedFromMqtt(client mqtt.Client, message mqtt.Message) {
	fmt.Printf("Received message on topic from mqtt: %s\n内容为: %s\n", message.Topic(), message.Payload())
	this.aliveCountMux.Lock()
	defer this.aliveCountMux.Unlock()
	this.aliveCount--
}

func (this *mqttDaemon)OnConnectHandler(client mqtt.Client){
	if token := client.Subscribe(*topic, byte(*qos), this.onMessageReceivedFromMqtt); token.Wait() && token.Error() != nil {
		fmt.Println("订阅topic: " + *topic + "失败。" + token.Error().Error())
	}else{
		fmt.Println("订阅topic: " + *topic + "成功")
		this.aliveCountMux.Lock()
		defer this.aliveCountMux.Unlock()
		this.aliveCount = 0
	}
}

func (this *mqttDaemon)ConnectionLostHandler(client mqtt.Client, err error){
	log.Println(time.Now(), "连接丢失" + err.Error())
	this.aliveCountMux.Lock()
	defer this.aliveCountMux.Unlock()
	this.aliveCount = 999
}

func(this *mqttDaemon)conn(){
	if token := this.client.Connect(); token.Wait() && token.Error() != nil {
		log.Println("连接失败", time.Now(), token.Error())
		panic("初次连接失败，禁止程序继续运行！")
	} else {
		fmt.Printf("Connected to %s\n", *mqttPort)
	}
}

func (this *mqttDaemon)updateAlive(){
	token :=this.client.Publish(*topic, 1, false, "alive")
	if(token.Error() != nil){
		fmt.Println(token.Error())
	}
	this.aliveCountMux.Lock()
	defer this.aliveCountMux.Unlock()
	this.aliveCount++
	fmt.Println(time.Now(), "updateAlive: " ,this.aliveCount)
}

func (this* mqttDaemon)checkAlive()(bool){
	fmt.Println(time.Now(), "aliveCount: " , this.aliveCount)
	this.aliveCountMux.Lock()
	defer this.aliveCountMux.Unlock()
	return this.aliveCount >3
}

func execCommand(commandName string, arg ...string) bool {
	cmd := exec.Command(commandName, arg...)
	//显示运行的命令
	fmt.Println(cmd.Args)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		fmt.Println(err)
		return false
	}

	cmd.Start()

	reader := bufio.NewReader(stdout)

	//实时循环读取输出流中的一行内容
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		fmt.Println(line)
	}

	cmd.Wait()
	return true
}

func (this* mqttDaemon)StartMqtt()(){
	this.aliveCountMux.Lock()
	defer this.aliveCountMux.Unlock()
	this.aliveCount = 0;
	//启动mqtt程序
	log.Println(time.Now(), "重启mqtt服务")
	execCommand("/bin/bash", "-c", *emqttShellPath + " start")
	log.Println(time.Now(), "重启mqtt服务完成")
}


func (this* mqttDaemon)StopMqtt()(){
	log.Println(time.Now(), "结束mqtt服务")
	execCommand("/bin/bash", "-c", *emqttShellPath + " stop")
	log.Println(time.Now(), "结束mqtt服务完成")
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	var damon mqttDaemon
	damon.init()
	//死循环更新数据
	for {
		time.Sleep(3000 * time.Millisecond)
		if(damon.checkAlive()){
			damon.StopMqtt()
			damon.StartMqtt()
		}
	}

}
