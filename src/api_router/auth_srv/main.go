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
	"../base/utils"
	"io/ioutil"
	"crypto/sha512"
	"crypto"
	"strings"
	"errors"
)

const AuthSrvName = "auth"
const AuthSrvVersion = "v1"
const (
	GateWayAddr = "127.0.0.1:8081"
	SrvAddr = "127.0.0.1:8001"
)

var g_apisMap = make(map[string]service.CallNodeApi)

// 注册方法
func callAuthFunction(req *data.SrvDispatchData, ack *data.SrvDispatchAckData) error{
	var err error
	h := g_apisMap[strings.ToLower(req.SrvArgv.Function)]
	if h != nil {
		err = h(req, ack)
	}else{
		err = errors.New("not find api")
	}

	if err != nil {
		ack.SrvAck.Err = data.ErrAuthSrvIllegalData
		ack.SrvAck.ErrMsg = data.ErrAuthSrvIllegalDataText
	}

	//fmt.Println("callNodeApi req: ", *req)
	//fmt.Println("callNodeApi ack: ", *ack)

	return err
}

func main() {
	wg := &sync.WaitGroup{}

	handler.AuthInstance().Init()

	// 启动db
	db.Init()

	// 创建节点
	nodeInstance, _:= service.NewServiceNode(AuthSrvName, AuthSrvVersion)
	nodeInstance.RegisterData.Addr = SrvAddr
	handler.AuthInstance().RegisterApi(&nodeInstance.RegisterData.Functions, &g_apisMap)
	nodeInstance.Handler = callAuthFunction

	nodeInstance.ServiceCenterAddr = GateWayAddr
	rpc.Register(nodeInstance)

	// 启动节点服务
	ctx, cancel := context.WithCancel(context.Background())
	nodeInstance.Start(ctx, wg)

	var err error
	var cipherData []byte

	time.Sleep(time.Second*2)
	for ; ;  {
		fmt.Println("Input 'quit' to quit...")
		var input string
		fmt.Scanln(&input)

		if input == "quit" {
			cancel()
			break;
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

			var data []byte
			for i := 0; i < 1024; i++ {
				data = append(data, byte(i))
			}

			fmt.Println("b:", time.Now())
			for i := 0; i < 1; i++ {
				fmt.Println("原始数据：", len(data))
				fmt.Println(data)
				// en
				cipherData, err = utils.RsaEncrypt(data, pubKey, utils.RsaEncodeLimit2048)
				if err != nil {
					fmt.Println(err)
					continue
				}

				fmt.Println("加密后数据：", len(cipherData))
				fmt.Println(cipherData)

				// de
				var originData []byte
				originData, err = utils.RsaDecrypt(cipherData, priKey, utils.RsaDecodeLimit2048)
				if err != nil {
					fmt.Println(err)
					continue
				}

				fmt.Println("解密后数据：")
				fmt.Println(originData)
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
			hashData = hs.Sum(nil)
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