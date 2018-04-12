package handler

import (
	"fmt"
	"../db"
	"../../base/utils"
	"../../data"
	"../../base/service"
	"crypto/sha512"
	"crypto"
	"io/ioutil"
	"encoding/base64"
	"sync"
	"errors"
	"../../account_srv/user"
	l4g "github.com/alecthomas/log4go"
)

type Auth struct{
	privateKey []byte

	rwmu sync.RWMutex
	usersLicenseKey map[string]*user.UserLevel
}

var defaultAuth = &Auth{}

func AuthInstance() *Auth{
	return defaultAuth
}

func (auth * Auth)Init(dir string) {
	var err error
	auth.privateKey, err = ioutil.ReadFile(dir+"/private.pem")
	if err != nil {
		l4g.Crash(err)
	}

	auth.usersLicenseKey = make(map[string]*user.UserLevel)
}

func (auth * Auth)getUserLevel(licenseKey string) (*user.UserLevel, error)  {
	ul := func() *user.UserLevel{
		auth.rwmu.RLock()
		defer auth.rwmu.RUnlock()

		return auth.usersLicenseKey[licenseKey]
	}()
	if ul != nil {
		return ul,nil
	}

	return func() (*user.UserLevel, error){
		auth.rwmu.Lock()
		defer auth.rwmu.Unlock()

		ul := auth.usersLicenseKey[licenseKey]
		if ul != nil {
			return ul, nil
		}
		ul, err := db.ReadUserLevel(licenseKey)
		if err != nil {
			return nil, err
		}
		return ul, nil
	}()
}

func (auth * Auth)GetApiGroup()(map[string]service.NodeApi){
	nam := make(map[string]service.NodeApi)

	apiInfo := data.ApiInfo{Name:"authdata", Level:data.APILevel_client}
	nam[apiInfo.Name] = service.NodeApi{ApiHandler:auth.AuthData, ApiInfo:apiInfo}

	apiInfo = data.ApiInfo{Name:"encryptdata", Level:data.APILevel_client}
	nam[apiInfo.Name] = service.NodeApi{ApiHandler:auth.EncryptData, ApiInfo:apiInfo}

	return nam
}

// 验证数据
func (auth *Auth)AuthData(req *data.SrvRequestData, res *data.SrvResponseData) {
	err := func() error{
		ul, err := auth.getUserLevel(req.Data.Argv.LicenseKey)
		if err != nil {
			l4g.Error("(%s) failed: %s",req.Data.Argv.LicenseKey, err.Error())
			return err
		}

		if req.Context.ApiLever > ul.Level || ul.IsFrozen != 0{
			l4g.Error("(%s-%s) failed: no api level or frozen", req.Data.Argv.LicenseKey, req.Data.Method.Function)
			return errors.New("no api level or frozen")
		}

		bencrypted, err := base64.StdEncoding.DecodeString(req.Data.Argv.Message)
		if err != nil {
			l4g.Error("%s", err.Error())
			return err
		}

		bsignature, err := base64.StdEncoding.DecodeString(req.Data.Argv.Signature)
		if err != nil {
			l4g.Error("%s", err.Error())
			return err
		}

		// 验证签名
		var hashData []byte
		hs := sha512.New()
		hs.Write(bencrypted)
		hashData = hs.Sum(nil)

		err = utils.RsaVerify(crypto.SHA512, hashData, bsignature, []byte(ul.PublicKey))
		if err != nil {
			l4g.Error("%s", err.Error())
			return err
		}

		// 解密数据
		var originData []byte
		originData, err = utils.RsaDecrypt(bencrypted, auth.privateKey, utils.RsaDecodeLimit2048)
		if err != nil {
			l4g.Error("%s", err.Error())
			return err
		}

		res.Data.Value.Message = string(originData)
		res.Data.Value.Signature = ""
		res.Data.Value.LicenseKey = req.Data.Argv.LicenseKey

		return nil
	}()

	if err != nil {
		res.Data.Err = data.ErrAuthSrvIllegalData
		res.Data.ErrMsg = data.ErrAuthSrvIllegalDataText
	}
}

// 打包数据
func (auth *Auth)EncryptData(req *data.SrvRequestData, res *data.SrvResponseData) {
	err := func() error{
		ul, err := auth.getUserLevel(req.Data.Argv.LicenseKey)
		if err != nil {
			l4g.Error("%s", err.Error())
			return err
		}

		// 加密数据不需要判断权限
		/*
		if req.Context.Api.Level > ul.Level || ul.IsFrozen != 0{
			fmt.Println("#Error AuthData--", err.Error())
			return errors.New("没权限或者被冻结")
		}*/

		// 加密
		bencrypted, err := func() ([]byte, error){
			// 用用户的pub加密message ->encrypteddata
			bencrypted, err := utils.RsaEncrypt([]byte(req.Data.Argv.Message), []byte(ul.PublicKey), utils.RsaEncodeLimit2048)
			if err != nil {
				return nil, err
			}

			return bencrypted, nil
		}()
		if err != nil {
			return err
		}
		res.Data.Value.Message = base64.StdEncoding.EncodeToString(bencrypted)

		// 签名
		bsignature, err := func() ([]byte, error){
			// 用服务器的pri签名encrypteddata ->signature
			var hashData []byte
			hs := sha512.New()
			hs.Write(bencrypted)
			hashData = hs.Sum(nil)

			bsignature, err := utils.RsaSign(crypto.SHA512, hashData, auth.privateKey)
			if err != nil {
				fmt.Println(err)
				return nil, err
			}

			return bsignature, nil
		}()
		if err != nil {
			return err
		}
		res.Data.Value.Signature = base64.StdEncoding.EncodeToString(bsignature)

		// licensekey
		res.Data.Value.LicenseKey = req.Data.Argv.LicenseKey

		return nil
	}()

	if err != nil {
		res.Data.Err = data.ErrAuthSrvIllegalData
		res.Data.ErrMsg = data.ErrAuthSrvIllegalDataText
	}
}