package main

import (
	"net/rpc"
	"../base/service"
	"../data"
	"./handler"
	"./db"
	"fmt"
	"context"
	"time"
	"sync"
	"../utils"
	"io/ioutil"
	"crypto/sha512"
	"crypto"
	"strings"
)

const AuthSrvName = "auth"
const AuthSrvVersion = "v1"
const (
	GateWayAddr = "127.0.0.1:8081"
	SrvAddr = "127.0.0.1:8091"
)

// 注册方法
func callAuthFunction(req *data.ServiceCenterDispatchData, ack *data.ServiceCenterDispatchAckData){
	var err error
	switch strings.ToLower(req.Function) {
	case "authdata":
		err = handler.AuthInstance().AuthData(req, ack)
		break
	case "encryptdata":
		err = handler.AuthInstance().EncryptData(req, ack)
		break
	}

	if err != nil {
		ack.Err = data.ErrAuthSrvIllegalData
		ack.ErrMsg = data.ErrAuthSrvIllegalDataText
	}

	fmt.Println("callNodeApi req: ", *req)
	fmt.Println("callNodeApi ack: ", *ack)
}

func main() {
	wg := &sync.WaitGroup{}

	handler.AuthInstance().Init()

	// 启动db
	db.Init()

	// 创建节点
	nodeInstance, _:= service.NewServiceNode(AuthSrvName, AuthSrvVersion)
	nodeInstance.RegisterData.Addr = SrvAddr
	nodeInstance.RegisterData.RegisterFunction(new(handler.Auth))
	nodeInstance.Handler = callAuthFunction

	nodeInstance.ServiceCenterAddr = GateWayAddr
	rpc.Register(nodeInstance)

	// 启动节点服务
	ctx, cancel := context.WithCancel(context.Background())
	nodeInstance.Start(ctx, wg)

	var err error
	var cipherData []byte

	var index int

	time.Sleep(time.Second*2)
	for ; ;  {
		fmt.Println("Input 'quit' to quit...")
		var input string
		fmt.Scanln(&input)

		if input == "quit" {
			cancel()
			break;
		}else if input == "rsagen"{
			index++
			pri := fmt.Sprintf("/Users/henly.liu/workspace/private_%d.pem", index)
			pub := fmt.Sprintf("/Users/henly.liu/workspace/public_%d.pem", index)
			err := utils.RsaGen(2048, pri, pub)
			if err != nil {
				fmt.Println(err)
				continue
			}

			pubKey, err := ioutil.ReadFile(pub)
			if err != nil {
				fmt.Println(err)
				continue
			}

			licenseKey := fmt.Sprintf("licensekey_%d", index)
			userName := fmt.Sprintf("username_%d", index)

			err = handler.AuthInstance().CreateUser(licenseKey, userName, string(pubKey))
			if err != nil {
				fmt.Println(err)
				continue
			}

			fmt.Println("rsagen ok")
		}else if input == "rsatest" {

			var priKey, pubKey []byte
			priKey, err = ioutil.ReadFile("/Users/henly.liu/workspace/private.pem")
			if err != nil {
				fmt.Println(err)
				continue
			}
			pubKey, err = ioutil.ReadFile("/Users/henly.liu/workspace/public.pem")
			if err != nil {
				fmt.Println(err)
				continue
			}

			fmt.Println("b:", time.Now())
			for i := 0; i < 20000; i++ {
				// en
				cipherData, err = utils.RsaEncrypt([]byte("123456"), pubKey)
				if err != nil {
					fmt.Println(err)
					continue
				}

				//fmt.Println("加密后数据：")
				//fmt.Println(cipherData)

				// de
				//var originData []byte
				_, err = utils.RsaDecrypt(cipherData, priKey)
				if err != nil {
					fmt.Println(err)
					continue
				}

				//fmt.Println("解密后数据：")
				//fmt.Println(string(originData))
			}
			fmt.Println("e:", time.Now())

		}else if input == "rsatest2" {
			var priKey, pubKey []byte
			priKey, err = ioutil.ReadFile("/Users/henly.liu/workspace/private.pem")
			if err != nil {
				fmt.Println(err)
				continue
			}
			pubKey, err = ioutil.ReadFile("/Users/henly.liu/workspace/public.pem")
			if err != nil {
				fmt.Println(err)
				continue
			}

			// sign
			var hashData []byte
			hs := sha512.New()
			hs.Write(cipherData)
			hashData = sha512.New().Sum(nil)
			fmt.Println("哈希后数据：")
			fmt.Println(hashData)

			var signData []byte
			signData, err = utils.RsaSign(crypto.SHA512, hashData, priKey)
			if err != nil {
				fmt.Println(err)
				continue
			}

			fmt.Println("签名后数据：")
			fmt.Println(signData)

			// verify
			err = utils.RsaVerify(crypto.SHA512, hashData, signData, pubKey)
			if err != nil {
				fmt.Println(err)
				continue
			}

			fmt.Println("验证后数据：")
			fmt.Println(err)
		}
	}

	fmt.Println("Waiting all routine quit...")
	wg.Wait()
	fmt.Println("All routine is quit...")
}