/*
Copyright Medcl (m AT medcl.net)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"github.com/paulbellamy/ratecounter"
	"infini.sh/framework/core/util"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

func WriteToLogstash(c *Config)  {
	if c.LogstashEndpoint==""{
		fmt.Println("logstash endpoint not defined")
		return
	}

	//go StartServer("127.0.0.1:5055")

	// connect to this socket

	//read files





	//fmt.Println(len(arr))
	//fmt.Println(arr[0])

	fmt.Println(c.Workers)
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}
	counter := ratecounter.NewRateCounter(1 * time.Second)
	for i:=0;i<c.Workers;i++{
		go func() {

			conn, err := net.Dial("tcp", c.LogstashEndpoint)
			if err != nil {
				fmt.Println(err)
				return
			}

			if c.LogstashSecEndpoint{
				conn, err = tls.Dial("tcp", c.LogstashEndpoint, conf)
				if err != nil {
					fmt.Println(err)
					return
				}
			}

			defer conn.Close()

			fmt.Println("worker:"+strconv.Itoa(i))

			if c.DumpInputFile!="" {
				f, err := os.Open(c.DumpInputFile)
				if err != nil {
					fmt.Println(err)
					return
				}

				//get file lines
				defer f.Close()
				r := bufio.NewReader(f)
				b, err := ioutil.ReadAll(r)
				if err != nil {
					log.Fatal(err)
				}

				var arr []map[string]interface{}
				_ = DecodeJson(string(b), &arr)


				for {
					//"13.64.105.169:5055"
					// read in input from stdin
					//reader := bufio.NewReader(os.Stdin)
					//fmt.Print("Text to send: ")
					//text, _ := reader.ReadString('\n')
					// send to socket

					for i,v:=range arr{
						//text:="{\"a\":\"123123\"}"
						obj:=map[string]interface{}{}
						//obj["level"]=c.TestLevel
						//obj["environment"]=c.TestEnvironment
						obj["i"]=i
						//fmt.Println(c.TestLevel)
						v["labels"]=obj


						_, e := conn.Write(util.ToJSONBytes(v))
						conn.Write([]byte("\r\n"))
						//fmt.Print(".")

						if e != nil {
							fmt.Println("Error to send message because of ", e.Error())
							break
						}
					}


					//fmt.Fprintf(conn, text + "\n")
					// listen for reply
					//message, _ := bufio.NewReader(conn).ReadString('\n')
					//fmt.Print("Message from server: "+message)
				}
			}else{

				for x:=0;x<10000000 ;x++  {
					v:="{\"a\":\"123123\"}"
					_, e := conn.Write([]byte(v))
					conn.Write([]byte("\r\n"))
					//fmt.Print(".")

					if e != nil {
						fmt.Println("Error to send message because of ", e.Error())
						break
					}
				}

			}



		}()

	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Type to Stop")
	reader.ReadString('\n')
	fmt.Println(counter.Rate())
}

const (
	BYTES_SIZE uint16 = 1024
	HEAD_SIZE  int    = 2
)

func StartServer(address string) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Println("Error listening", err.Error())
		return
	}
	for {
		conn, err := listener.Accept()
		fmt.Println(conn.RemoteAddr())
		if err != nil {
			fmt.Println("Error accepting", err.Error())
			return // 终止程序
		}
		go doConn(conn)
	}
}

func doConn(conn net.Conn) {
	var (
		buffer      = newBuffer(conn, 16)
		headBuf     []byte
		contentSize int
		contentBuf  []byte
	)
	for {
		_, err := buffer.readFromReader()
		if err != nil {
			fmt.Println(err)
			return
		}
		for {
			headBuf, err = buffer.seek(HEAD_SIZE)
			if err != nil {
				break
			}
			contentSize = int(binary.BigEndian.Uint16(headBuf))
			if buffer.Len() >= contentSize-HEAD_SIZE {
				contentBuf = buffer.read(HEAD_SIZE, contentSize)
				fmt.Println(string(contentBuf))
				continue
			}
			break
		}
	}
}
