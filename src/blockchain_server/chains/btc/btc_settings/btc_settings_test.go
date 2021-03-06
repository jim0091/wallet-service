package btc_settings

import (
	L4g "blockchain_server/L4g"
	"testing"
)

var (
	key_settings *KeySettings
	rpc_settings *RPCSettings
)

func TestSettings  (t *testing.T) {
	//if err := load_keysettings(); err!=nil {
	//	// TODO: should return ???? or create new master key???
	//	L4g.Error("btc load keysettings err message, %s", err.Error())
	//	if !config.Debugmode { return }
	//}

	btcconfig := Client_config()
	if nil==btcconfig { return }

	if Key_settings, err := KeySettings_from_MainConfig(); err!=nil {

	} else if Key_settings ==nil || !Key_settings.IsValid() {
		if err:=initMajorkey(); err!=nil {
			L4g.Error("HDWallet init error, message:%s", err.Error())
		}
	}

	//if rpc_settings, err := RPCSettings_from_MainConfig(); err!=nil {
	//	L4g.Error("rpc settings faild, message:%s", err.Error())
	//}

	btcconfig.SubConfigs[Name_KeySettings] = key_settings
	btcconfig.SubConfigs[Name_RPCSettings] = rpc_settings

	btcconfig.Save()

	L4g.Trace("BTC client init ok!")
}

