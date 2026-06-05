package service

import (
	"encoding/json"
	"io"
	"regexp"
	"strings"
)

type IdentityStruct struct {
	Role           string
	RoleName       string
	UserCategoryID string
}

func GetIdentity(UserName, PassWord string) string {
	return GetIdentityWithLogin(UserName, PassWord, "", "")
}

func GetIdentityWithLogin(UserName, PassWord, loginType, authServerURL string) string {
	result := IdentityResult{Type: "identity"}
	conf := ReadConfig()
	client, err := CreateLoggedInClient(conf, UserName, PassWord, LoginOptions{LoginType: loginType, AuthServerURL: authServerURL})
	if err != nil {
		result.Data = IdentityStruct{Role: "unknown", RoleName: "未知"}
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}

	response, err := client.Get(conf.MangerURL + "eams/homeExt.action")
	if err != nil {
		result.Data = IdentityStruct{Role: "unknown", RoleName: "未知"}
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	defer response.Body.Close()
	content, _ := io.ReadAll(response.Body)
	result.Data = ParseHomeExtIdentity(string(content))
	js, _ := json.MarshalIndent(result, "", "\t")
	return B2S(js)
}

func ParseHomeExtIdentity(html string) IdentityStruct {
	identity := IdentityStruct{Role: "unknown", RoleName: "未知"}
	categoryReg := regexp.MustCompile(`(?is)<input[^>]+name=["']security\.userCategoryId["'][^>]*value=["']([^"']+)["']`)
	if match := categoryReg.FindStringSubmatch(html); len(match) > 1 {
		identity.UserCategoryID = strings.TrimSpace(match[1])
		switch identity.UserCategoryID {
		case "1":
			identity.Role = "student"
			identity.RoleName = "学生"
			return identity
		case "2":
			identity.Role = "teacher"
			identity.RoleName = "教师"
			return identity
		}
	}

	if strings.Contains(html, "courseTableForStd.action") || strings.Contains(html, "stdDetail.action") || strings.Contains(html, "学生") {
		identity.Role = "student"
		identity.RoleName = "学生"
		return identity
	}
	if strings.Contains(html, "courseTableForTeacher.action") || strings.Contains(html, "teacherExamTable.action") || strings.Contains(html, "教师") {
		identity.Role = "teacher"
		identity.RoleName = "教师"
		return identity
	}
	return identity
}
