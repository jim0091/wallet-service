package address

import (
	"blockchain_server/service"
	"blockchain_server/types"
	. "business_center/def"
	"business_center/mysqlpool"
	"business_center/redispool"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Address struct {
	wallet           *service.ClientManager
	mapUserProperty  map[string]*UserProperty
	mapAssetProperty map[string]*AssetProperty
	mapUserAddress   map[string]*UserAddress
	rechargeChannel  types.RechargeTxChannel
	cmdTxChannel     types.CmdTxChannel
	waitGroup        sync.WaitGroup
	ctx              context.Context
}

func (a *Address) Run(ctx context.Context, wallet *service.ClientManager) {
	a.wallet = wallet
	a.ctx = ctx

	a.mapUserProperty, _ = mysqlpool.QueryAllUserProperty()
	a.mapAssetProperty, _ = mysqlpool.QueryAllAssetProperty()
	a.mapUserAddress, _ = mysqlpool.QueryAllUserAddress()

	a.rechargeChannel = make(types.RechargeTxChannel)
	a.cmdTxChannel = make(types.CmdTxChannel)

	a.recvRechargeTxChannel()
	a.recvCmdTxChannel()

	a.wallet.SubscribeTxRecharge(a.rechargeChannel)
	a.wallet.SubscribeTxCmdState(a.cmdTxChannel)

	//添加监控地址
	for _, v := range a.mapUserAddress {
		rcaCmd := service.NewRechargeAddressCmd("", v.AssetName, []string{v.Address})
		a.wallet.InsertRechargeAddress(rcaCmd)
	}
}

func (a *Address) Stop() {
	a.waitGroup.Wait()
}

func (a *Address) AllocationAddress(req string, ack *string) error {
	var reqInfo ReqNewAddress
	err := json.Unmarshal([]byte(req), &reqInfo)
	if err != nil {
		fmt.Printf("AllocationAddress Unmarshal Error : %s/n", err.Error())
		return err
	}

	var rspInfo RspNewAddress
	rspInfo.Result.ID = reqInfo.UserID
	rspInfo.Result.Symbol = reqInfo.Params.Symbol
	rspInfo.Status.Code = 0
	rspInfo.Status.Msg = ""

	userProperty, ok := a.mapUserProperty[reqInfo.UserID]
	if !ok {
		return errors.New("AllocationAddress mapUserProperty find Error")
	}

	assetProperty, ok := a.mapAssetProperty[reqInfo.Params.Symbol]
	if !ok {
		return errors.New("AllocationAddress mapAssetProperty find Error")
	}

	mapUserAddress, _ := a.generateAddress(userProperty.UserID, userProperty.UserClass, assetProperty.ID,
		assetProperty.Name, reqInfo.Params.Count)
	if len(mapUserAddress) > 0 {
		rspInfo.Result.Address = a.addUserAddress(mapUserAddress)

		//添加监控地址
		rcaCmd := service.NewRechargeAddressCmd("message id", assetProperty.Name, rspInfo.Result.Address)
		a.wallet.InsertRechargeAddress(rcaCmd)
	}

	pack, err := json.Marshal(rspInfo)
	if err != nil {
		fmt.Printf("AllocationAddress RspNewAddress Marshal Error : %s/n", err.Error())
		return err
	}
	*ack = string(pack)
	return nil
}

func (a *Address) generateAddress(userID string, userClass int,
	assetID int, assetName string, count int) (map[string]*UserAddress, error) {
	mapUserAddress := make(map[string]*UserAddress)
	cmd := service.NewAccountCmd("", assetName, 1)

	for i := 0; i < count; i++ {
		accounts, err := a.wallet.NewAccounts(cmd)
		if err != nil {
			fmt.Printf("generateAddress NewAccounts Error : %s\n", err.Error())
			return mapUserAddress, err
		}
		userAddress := &UserAddress{}
		userAddress.UserID = userID
		userAddress.UserClass = userClass
		userAddress.AssetID = assetID
		userAddress.AssetName = assetName
		userAddress.Address = accounts[0].Address
		userAddress.PrivateKey = accounts[0].PrivateKey
		userAddress.AvailableAmount = 0
		userAddress.FrozenAmount = 0
		userAddress.Enabled = 1
		userAddress.CreateTime = time.Now().Unix()
		userAddress.UpdateTime = time.Now().Unix()

		mapUserAddress[assetName+"_"+accounts[0].Address] = userAddress
	}
	return mapUserAddress, nil
}

func (a *Address) addUserAddress(mapUserAddress map[string]*UserAddress) []string {
	var addresses []string
	tx, err := mysqlpool.Get().Begin()
	if err != nil {
		return addresses
	}

	for k, v := range mapUserAddress {
		_, err := tx.Exec("insert user_address (user_id, asset_id, address, private_key, available_amount, frozen_amount, "+
			"enabled, create_time, update_time) values (?, ?, ?, ?, ?, ?, ?, ?, ?);",
			v.UserID, v.AssetID, v.Address, v.PrivateKey, v.AvailableAmount, v.FrozenAmount, v.Enabled,
			time.Unix(v.CreateTime, 0).UTC().Format("2006-01-02 15:04:05"),
			time.Unix(v.UpdateTime, 0).UTC().Format("2006-01-02 15:04:05"))
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		a.mapUserAddress[k] = v
		addresses = append(addresses, v.Address)
	}
	tx.Commit()
	return addresses
}

func (a *Address) recvRechargeTxChannel() {
	a.waitGroup.Add(1)
	go func(ctx context.Context, channel types.RechargeTxChannel) {
		c := redispool.Get()
		defer c.Close()

		for {
			select {
			case rct := <-channel:
				{
					assetProperty, ok := a.mapAssetProperty[rct.Coin_name]
					if !ok {
						continue
					}

					switch rct.Tx.State {
					case types.Tx_state_mined: //入块
						{
							var blockin TransactionBlockin
							blockin.AssetID = assetProperty.ID
							blockin.Hash = rct.Tx.Tx_hash
							blockin.AssetName = assetProperty.Name
							blockin.BlockinHeight = rct.Tx.InBlock
							blockin.BlockinTime = int64(rct.Tx.Time)
							blockin.OrderID = ""

							a.transactionBegin(&blockin)
						}
					case types.Tx_state_commited: //确认
						{
							var state TransactionStatus
							state.AssetID = assetProperty.ID
							state.Hash = rct.Tx.Tx_hash
							state.AssetName = assetProperty.Name
							state.Status = 1
							state.ConfirmHeight = rct.Tx.ConfirmatedHeight
							state.ConfirmTime = int64(rct.Tx.Time)
							state.UpdateTime = time.Now().Unix()

							a.transactionFinish(&state)
						}
					case types.Tx_state_unconfirmed: //失败
						{
							var state TransactionStatus
							state.AssetID = assetProperty.ID
							state.Hash = rct.Tx.Tx_hash
							state.AssetName = assetProperty.Name
							state.Status = 2
							state.ConfirmHeight = rct.Tx.ConfirmatedHeight
							state.ConfirmTime = int64(rct.Tx.Time)
							state.UpdateTime = time.Now().Unix()

							a.transactionFinish(&state)
						}
					}
				}
			case <-ctx.Done():
				{
					fmt.Println("RechangeTx context done, because : ", ctx.Err())
					a.waitGroup.Done()
					return
				}
			}
		}
	}(a.ctx, a.rechargeChannel)
}

func (a *Address) recvCmdTxChannel() {
	a.waitGroup.Add(1)
	go func(ctx context.Context, channel types.CmdTxChannel) {
		for {
			select {
			case cmdTx := <-channel:
				{
					assetProperty, ok := a.mapAssetProperty[cmdTx.Coinname]
					if !ok {
						continue
					}

					switch cmdTx.Tx.State {
					case types.Tx_state_mined: //入块
						{
							var blockin TransactionBlockin
							blockin.AssetID = assetProperty.ID
							blockin.Hash = cmdTx.Tx.Tx_hash
							blockin.AssetName = assetProperty.Name
							blockin.BlockinHeight = cmdTx.Tx.InBlock
							blockin.BlockinTime = int64(cmdTx.Tx.Time)
							blockin.OrderID = cmdTx.NetCmd.MsgId

							a.transactionBegin(&blockin)
						}
					case types.Tx_state_commited: //确认
						{
							var state TransactionStatus
							state.AssetID = assetProperty.ID
							state.Hash = cmdTx.Tx.Tx_hash
							state.AssetName = assetProperty.Name
							state.Status = 1
							state.ConfirmHeight = cmdTx.Tx.ConfirmatedHeight
							state.ConfirmTime = int64(cmdTx.Tx.Time)
							state.UpdateTime = time.Now().Unix()

							a.transactionFinish(&state)
						}
					case types.Tx_state_unconfirmed: //失败
						{
							var state TransactionStatus
							state.AssetID = assetProperty.ID
							state.Hash = cmdTx.Tx.Tx_hash
							state.AssetName = assetProperty.Name
							state.Status = 2
							state.ConfirmHeight = cmdTx.Tx.ConfirmatedHeight
							state.ConfirmTime = int64(cmdTx.Tx.Time)
							state.UpdateTime = time.Now().Unix()

							a.transactionFinish(&state)
						}
					}
				}
			case <-ctx.Done():
				fmt.Println("TxState context done, because : ", ctx.Err())
				a.waitGroup.Done()
				return
			}
		}
	}(a.ctx, a.cmdTxChannel)
}

func (a *Address) transactionBegin(blockin *TransactionBlockin) {
	fmt.Println(blockin)
}

func (a *Address) transactionFinish(status *TransactionStatus) {
	fmt.Println(status)
}
