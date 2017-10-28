package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"
)

var mqttPort = flag.String("mqttPort", "61613", "mqttPort of broker")
var qos = flag.Int("qos", 0, "The QoS to subscribe to messages at")
var clientid = flag.String("clientid", "NodeHopeMqttDaemon", "A clientid for the connection")
var user = flag.String("user", "wifi", "username")
var pass = flag.String("pass", "68008232", "password")
var topic = flag.String("topic", "mqttdaemon", "topic")
//var emqttShellPath = flag.String("emqttShellPath", "/Users/kitty/Downloads/emqttd/bin/emqttd", "emqtt shell")
//execCommand("/bin/bash", "-c", *emqttShellPath+" start")

type mqttDaemon struct {
	aliveCount int64
	client     mqtt.Client

	aliveCountMux sync.Mutex
}


func (this *mqttDaemon) init() {
	connOpts := &mqtt.ClientOptions{
		ClientID:             *clientid,
		CleanSession:         true,
		Username:             *user,
		Password:             *pass,
		ConnectTimeout:       3 * time.Second,
		AutoReconnect:        true,
		MaxReconnectInterval: 3 * time.Second,
		KeepAlive:            30 * time.Second,
		OnConnect:            this.OnConnectHandler,
		OnConnectionLost:     this.ConnectionLostHandler,
		TLSConfig:            tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert},
	}
	connOpts.AddBroker("tcp://127.0.0.1:" + *mqttPort)
	this.client = mqtt.NewClient(connOpts)
	this.conn()
}

/*
从mqtt接受消息，并把消息转发到浏览器
*/
func (this *mqttDaemon) onMessageReceivedFromMqtt(client mqtt.Client, message mqtt.Message) {
	fmt.Printf("Received message on topic from mqtt: %s\n内容为: %s\n", message.Topic(), message.Payload())
}

func (this *mqttDaemon) OnConnectHandler(client mqtt.Client) {
	if token := client.Subscribe(*topic, byte(*qos), this.onMessageReceivedFromMqtt); token.Wait() && token.Error() != nil {
		fmt.Println("订阅topic: " + *topic + "失败。" + token.Error().Error())
	} else {
		fmt.Println("订阅topic: " + *topic + "成功")
	}
}

func (this *mqttDaemon) ConnectionLostHandler(client mqtt.Client, err error) {
	log.Println(time.Now(), "连接丢失"+err.Error())
	this.StartMqtt()
}

func (this *mqttDaemon) conn() {
	if token := this.client.Connect(); token.Wait() && token.Error() != nil {
		log.Println("初次连接失败, 启动mqtt", time.Now(), token.Error())
		this.StartMqtt()
		this.conn()
	} else {
		log.Println("Connected to %s\n", *mqttPort)
	}
}

func (this *mqttDaemon) updateAlive() {
	this.aliveCountMux.Lock()
	defer this.aliveCountMux.Unlock()
	if this.client.IsConnected(){
		this.client.Publish(*topic, 1, false, "alive")
	}
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

func (this *mqttDaemon) StartMqtt() {
	this.aliveCountMux.Lock()
	defer this.aliveCountMux.Unlock()
	//启动mqtt程序
	log.Println(time.Now(), "重启mqtt服务")
	execCommand("/bin/bash", "-c", "docker restart emq20")
	log.Println(time.Now(), "重启mqtt服务完成")
}

func main() {

	flag.Parse()
	log.SetFlags(0)
	var damon mqttDaemon
	damon.init()
	//死循环更新数据
	for {
		damon.updateAlive()
		time.Sleep(3000 * time.Millisecond)
	}
}

