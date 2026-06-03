package service

import "encoding/json"

func GetUserLogin(UserName, PassWord string) string {
	return GetUserLoginWithLogin(UserName, PassWord, "", "")
}

func GetUserLoginWithLogin(UserName, PassWord, loginType, authServerURL string) string {
	result := LoginResult{Type: "login"}
	conf := ReadConfig()
	client, err := CreateLoggedInClient(conf, UserName, PassWord, LoginOptions{LoginType: loginType, AuthServerURL: authServerURL})
	if err != nil {
		result.Data = "登录失败"
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	response, err := client.Get(conf.MangerURL + "eams/logout.action")
	if err == nil {
		defer response.Body.Close()
	}
	result.Data = "登录成功"
	js, _ := json.MarshalIndent(result, "", "\t")
	return B2S(js)
}
