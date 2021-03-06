package v1

import (
	"bastionpay_api/apidoc"
	"bastionpay_api/api/v1"
	"bastionpay_api/apibackend"
)

var ApiDocAdd = apidoc.ApiDoc{
	Group:[]string{apibackend.HttpRouterApi},
	VerName:"v1",
	SrvName:"arith",
	FuncName:"add",
	Input:&v1.Args{},
	Output:&v1.AckArgs{},
}