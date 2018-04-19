package business

import (
	"api_router/base/data"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"testing"
	"time"
)

func TestHandleMsg(t *testing.T) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"))

	svr := NewBusinessSvr()
	svr.InitAndStart(nil)
	var req data.SrvRequestData
	var res data.SrvResponseData

	testType := 3
	switch testType {
	case 1:
		req.Data.Method.Function = "new_address"
		req.Data.Argv.UserKey = "737205c4-af3c-426d-973d-165a0bf46c71"
		req.Data.Argv.Message = "{\"id\":\"1\",\"symbol\":\"eth\",\"count\":1}"
	case 2:
		req.Data.Method.Function = "withdrawal"
		req.Data.Argv.UserKey = "737205c4-af3c-426d-973d-165a0bf46c71"
		req.Data.Argv.Message = "{\"user_order_id\":\"1\",\"symbol\":\"eth\",\"amount\":0.1,\"to_address\":\"0x00000\",\"user_timestamp\":0}"
	case 3:
		req.Data.Method.Function = "query_user_address"
		req.Data.Argv.UserKey = "737205c4-af3c-426d-973d-165a0bf46c71"
		req.Data.Argv.Message = "{\"page_index\":1,\"max_display\":1,\"create_time_begin\":1523656800}"
	}

	if testType > 0 {
		svr.HandleMsg(&req, &res)
		fmt.Println(res.Data.Value.Message)
	}

	//s := "{\"user_id\":\"795b587d-2ee7-4979-832d-5d0ea64205d5\",\"method\":\"new_address\",\"params\":{\"id\":\"1\",\"symbol\":\"eth\",\"count\":1}}"
	//s := "{\"user_id\":\"737205c4-af3c-426d-973d-165a0bf46c71\",\"method\":\"withdrawal\",\"params\":{\"user_order_id\":\"1\",\"symbol\":\"eth\",\"amount\":0.1,\"to_address\":\"0x00000\",\"user_timestamp\":0}}"
	//var reply string
	//svr.HandleMsg(s, &reply)
	//fmt.Println(reply)

	time.Sleep(time.Second * 60 * 60)
	svr.Stop()
}
