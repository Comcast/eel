package jtl

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	. "github.com/Comcast/eel/util"
)

/*
  "ProfileServiceEP"          : "http://directory.g.comcast.net/profile-service/profile/",
  "CloudProfileServiceEP"     : "https://profile-service.apps.cloud.comcast.net/profile-service/profile/",
  "CloudProfileServiceEP_QA"  : "https://profile-service-qa.u1.app.cloud.comcast.net/profile-service/profile/",
  "CloudProfileServiceEP_INT" : "https://profile-service-int.u1.app.cloud.comcast.net/profile-service/profile/",
*/

type (
	ProfileServiceConf struct {
		ProfileServiceEP string `json:"ProfileServiceEP"`
		ClientId         string `json:"ClientId"`
		ClientKey        string `json:"ClientKey"`
	}
	ProfileServiceClient struct {
		ProfileServiceConf *ProfileServiceConf
	}
)

func NewProfileServiceClient(ctx Context) (*ProfileServiceClient, error) {
	profileServiceClient := &ProfileServiceClient{
		ProfileServiceConf: &ProfileServiceConf{},
	}

	err := profileServiceClient.LoadConfig(ctx, "config-eel/profile_service.json", profileServiceClient.ProfileServiceConf)
	if nil != err {
		ctx.Log().Error("op", "NewProfileServiceClient", "error", err)
		return nil, err
	}
	ctx.Log().Debug("op", "NewProfileServiceClient", "profileServiceClient", profileServiceClient)
	return profileServiceClient, nil
}

func (profileService *ProfileServiceClient) LoadConfig(ctx Context, filename string, conf *ProfileServiceConf) error {
	if _, err := os.Stat(filename); nil != err {
		return err
	}

	bs, err := ioutil.ReadFile(filename)
	if nil != err {
		return err
	}

	if err = json.Unmarshal(bs, conf); nil != err {
		return err
	}
	return nil
}

func calculateRFC2104HMAC(input, key string) string {
	key_for_sign := []byte(key)
	h := hmac.New(sha1.New, key_for_sign)
	h.Write([]byte(input))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (profileService *ProfileServiceClient) GetAuthHeaderByAccountNumber(accountNumber string, headers map[string]string) {

	currentTime := time.Now().Format("2006-01-02T15:04:05.000-0700")
	hmacData := profileService.ProfileServiceConf.ClientId + accountNumber + currentTime
	hmackey := calculateRFC2104HMAC(hmacData, profileService.ProfileServiceConf.ClientKey)

	headers["Authorization"] = profileService.ProfileServiceConf.ClientId + ":" + hmackey
	headers["Date"] = currentTime
	//compare to package jmcvetta/napping, http.Client doesn't set Accept header by default, which causes PS call fails
	headers["Accept"] = "*/*"

	return
}
