package address

import (
	"blockchain_server/service"
	"blockchain_server/types"
	"business_center/def"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"time"
)

type Address struct {
	wallet        *service.ClientManager
	symbolNameMap map[string]int
}

func (addr *Address) Init(wallet *service.ClientManager) {
	addr.wallet = wallet
	addr.symbolNameMap = make(map[string]int)
	addr.symbolNameMap[types.Chain_bitcoin] = 1
	addr.symbolNameMap[types.Chain_eth] = 2

	addr.loadAaddress()
}

func (addr *Address) loadAaddress() {
	c, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Printf("loadAaddress Redis Dial Error: %s", err.Error())
		return
	}
	defer c.Close()

	for symbolName := range addr.symbolNameMap {
		jsonInfo, err := redis.Strings(c.Do("hvals", "user_address_"+symbolName))
		if err != nil {
			return
		}

		var addrs []string
		for _, jsonInfo := range jsonInfo {
			var userAddr def.UserAddress
			json.Unmarshal([]byte(jsonInfo), &userAddr)
			addrs = append(addrs, userAddr.Address)
		}

		if len(addrs) > 0 {
			rcaCmd := types.NewRechargeAddressCmd("message id", symbolName, addrs)
			addr.wallet.InsertRechargeAddress(rcaCmd)
		}
	}
}

func (addr *Address) AllocationAddress(req string, ack *string) error {
	reqInfo := &def.ReqNewAddress{}
	err := json.Unmarshal([]byte(req), reqInfo)
	if err != nil {
		fmt.Printf("AllocationAddress Unmarshal Error : %s/n", err.Error())
		return err
	}

	c, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Printf("AllocationAddress Redis Dial Error: %s", err.Error())
		return err
	}
	defer c.Close()

	symbolID, ok := addr.symbolNameMap[reqInfo.Params.Symbol]
	if !ok {
		err := errors.New("AllocationAddress SymbolName Invalid")
		fmt.Println(err.Error())
		return err
	}
	symbolName := reqInfo.Params.Symbol

	//查询剩余的地址
	idleNumber, err := redis.Int(c.Do("scard", "free_address_"+symbolName))
	if err != nil {
		fmt.Printf("AllocationAddress Redis Scard Error: %s\n", err.Error())
		return err
	}

	//补充地址
	if idleNumber < 500 {
		err := addr.generateAddress(c, symbolName, 1000-idleNumber)
		if err != nil {
			return err
		}
	}

	accounts, err := redis.Strings(c.Do("spop", "free_address_"+symbolName, reqInfo.Params.Count))
	if err != nil {
		fmt.Printf("AllocationAddress Redis SPOP Error : %s/n", err.Error())
		return err
	}

	var addrs []string
	for _, v := range accounts {
		acc := &types.Account{}
		err := json.Unmarshal([]byte(v), acc)
		if err != nil {
			fmt.Printf("AllocationAddress Accounts Unmarshal Error : %s/n", err.Error())
			return nil
		}
		addrs = append(addrs, acc.Address)

		var userAddr def.UserAddress
		userAddr.UserID = reqInfo.UserID
		userAddr.AssetID = symbolID
		userAddr.Address = acc.Address
		userAddr.PrivateKey = acc.PrivateKey
		userAddr.Enabled = true
		userAddr.CreateTime = uint64(time.Now().Unix())
		jsonInfo, err := json.Marshal(userAddr)
		if err != nil {
			fmt.Printf("AllocationAddress UserAddress Marshal Error : %s/n", err.Error())
			return err
		}
		c.Do("hset", "user_address_"+symbolName, acc.Address, jsonInfo)
	}

	if len(addrs) > 0 {
		//创建资金帐户
		nowUnix := uint64(time.Now().Unix())
		acc := &def.UserAccount{}
		acc.UserID = reqInfo.UserID
		acc.AssetID = symbolID
		acc.AvailableAmount = 0
		acc.FrozenAmount = 0
		acc.CreateTime = nowUnix
		acc.UpdateTime = nowUnix
		jsonInfo, _ := json.Marshal(acc)

		for {
			c.Do("watch", "user_account")
			a, err := redis.Int(c.Do("hexists", "user_account", reqInfo.UserID+"_"+symbolName))
			if err != nil {
				return err
			}
			if a == 0 {
				c.Do("multi")
				c.Do("hset", "user_account", reqInfo.UserID+"_"+symbolName, jsonInfo)
				reply, err := c.Do("exec")
				if err != nil {
					return err
				}
				if reply != nil {
					break
				}
			} else {
				c.Do("unwatch", "user_account")
				break
			}
		}

		rcaCmd := types.NewRechargeAddressCmd("message id", symbolName, addrs)
		addr.wallet.InsertRechargeAddress(rcaCmd)
	}

	rsp := new(def.RspNewAddress)
	rsp.Result.ID = reqInfo.UserID
	rsp.Result.Symbol = reqInfo.Params.Symbol
	rsp.Result.Address = addrs
	rsp.Status.Code = 0
	rsp.Status.Msg = ""

	byteRsp, err := json.Marshal(rsp)
	if err != nil {
		fmt.Printf("AllocationAddress RspNewAddress Marshal Error : %s/n", err.Error())
		return err
	}

	*ack = string(byteRsp)
	return nil
}

func (addr *Address) generateAddress(c redis.Conn, symbolName string, count int) error {
	accCmd := types.NewAccountCmd("message id", symbolName, 1)

	for i := 0; i < count; i++ {
		accounts, err := addr.wallet.NewAccounts(accCmd)
		if err != nil {
			fmt.Printf("generateAddress NewAccounts Error : %s\n", err.Error())
			return err
		}
		jsonInfo, err := json.Marshal(accounts[0])
		if err != nil {
			fmt.Printf("generateAddress Marshal Error : %s\n", err.Error())
			return err
		}
		c.Do("sadd", "free_address_"+symbolName, jsonInfo)
	}
	return nil
}

//func (busi *Business) HandleWithdrawal(args string, replyChan chan string) error {
//	req := new(def.ReqWithdrawal)
//	err := json.Unmarshal([]byte(args), req)
//	if err != nil {
//		fmt.Printf("HandleWithdrawal Json Unmarshal Error:%s", err.Error())
//		return err
//	}
//
//	rsp := new(def.RspWithdrawal)
//	rsp.Result.UserOrderID = req.Params.UserOrderID
//	rsp.Result.Timestamp = "0"
//	rsp.Status.Code = 0
//	rsp.Status.Msg = ""
//
//	reply, _ := json.Marshal(rsp)
//
//	replyChan <- string(reply)
//
//	return nil
//}