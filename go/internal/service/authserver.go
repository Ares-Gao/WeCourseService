package service

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0"
const aesChars = "ABCDEFGHJKMNPQRSTWXYZabcdefhijkmnprstwxyz2345678"

type LoginOptions struct {
	LoginType     string
	AuthServerURL string
}

func CreateLoggedInClient(conf Config, username, password string, options LoginOptions) (*http.Client, error) {
	loginType := strings.ToLower(strings.TrimSpace(options.LoginType))
	if loginType == "" {
		loginType = strings.ToLower(strings.TrimSpace(conf.LoginType))
	}
	if loginType == "authserver" {
		loginURL := strings.TrimSpace(options.AuthServerURL)
		if loginURL == "" {
			loginURL = strings.TrimSpace(conf.AuthServerURL)
		}
		return createAuthServerLoggedInClient(conf, username, password, loginURL)
	}
	return createDirectLoggedInClient(conf, username, password)
}

func createDirectLoggedInClient(conf Config, username, password string) (*http.Client, error) {
	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Jar: cookieJar}

	response, err := client.Get(conf.MangerURL + "eams/login.action")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	html := string(content)
	index := strings.Index(html, "CryptoJS.SHA1(")
	if index == -1 || len(html) < index+52 {
		return nil, errors.New("password salt not found")
	}
	salt := html[index+15 : index+52]
	hash := sha1.Sum([]byte(salt + password))

	formValues := make(url.Values)
	formValues.Set("username", username)
	formValues.Set("password", hex.EncodeToString(hash[:]))
	formValues.Set("session_locale", "zh_CN")
	time.Sleep(time.Second)

	request, err := http.NewRequest(http.MethodPost, conf.MangerURL+"eams/login.action", strings.NewReader(formValues.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", userAgent)
	response, err = client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	content, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(string(content), "<a href=\"/eams/security/my.action\" target=\"_blank\" title=\"查看详情\" style=\"color:#ffffff\">") {
		return nil, errors.New("login failed")
	}
	return client, nil
}

func createAuthServerLoggedInClient(conf Config, username, password, loginURL string) (*http.Client, error) {
	if loginURL == "" {
		return nil, errors.New("AuthServerURL is required for authserver login")
	}
	retries := conf.AuthServerCaptchaRetries
	if retries <= 0 {
		retries = 1
	}

	var lastErr error
	for i := 0; i < retries; i++ {
		client, err := createAuthServerLoggedInClientOnce(conf, username, password, loginURL)
		if err == nil {
			return client, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func createAuthServerLoggedInClientOnce(conf Config, username, password, loginURL string) (*http.Client, error) {
	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Jar: cookieJar}

	response, err := client.Get(loginURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	html := string(content)

	salt := inputValue(html, "", "pwdEncryptSalt")
	execution := inputValue(html, "execution", "")
	if salt == "" || execution == "" {
		return nil, errors.New("authserver login page is missing pwdEncryptSalt or execution")
	}

	captcha := ""
	if conf.AuthServerAutoCaptcha && needAuthServerCaptcha(html) {
		captcha, err = recognizeAuthServerCaptcha(client, conf, response.Request.URL.String())
		if err != nil {
			return nil, err
		}
	}

	encryptedPassword, err := authServerEncryptPassword(password, salt)
	if err != nil {
		return nil, err
	}
	formValues := make(url.Values)
	formValues.Set("username", username)
	formValues.Set("password", encryptedPassword)
	formValues.Set("captcha", captcha)
	formValues.Set("_eventId", valueOr(inputValue(html, "_eventId", ""), "submit"))
	formValues.Set("cllt", "userNameLogin")
	formValues.Set("dllt", valueOr(inputValue(html, "dllt", ""), "generalLogin"))
	formValues.Set("lt", inputValue(html, "lt", ""))
	formValues.Set("execution", execution)
	formValues.Set("rmShown", "1")

	request, err := http.NewRequest(http.MethodPost, response.Request.URL.String(), strings.NewReader(formValues.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Referer", response.Request.URL.String())
	request.Header.Set("Origin", response.Request.URL.Scheme+"://"+response.Request.URL.Host)

	response, err = client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	content, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	finalPath := ""
	if response.Request != nil && response.Request.URL != nil {
		finalPath = response.Request.URL.Path
	}
	if strings.Contains(finalPath, "/authserver/login") || strings.Contains(string(content), "认证失败") || strings.Contains(strings.ToLower(string(content)), "captcha") {
		return nil, errors.New("authserver login failed")
	}
	return client, nil
}

func recognizeAuthServerCaptcha(client *http.Client, conf Config, loginURL string) (string, error) {
	authBase, err := authServerBase(loginURL)
	if err != nil {
		return "", err
	}
	response, err := client.Get(authBase + "/getCaptcha.htl?" + fmt.Sprint(time.Now().UnixMilli()))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	imageBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	result, err := SolveDdddOcrCaptcha(conf, imageBytes)
	if err != nil {
		return "", err
	}
	return regexp.MustCompile(`[^0-9A-Za-z]`).ReplaceAllString(result, ""), nil
}

func needAuthServerCaptcha(html string) bool {
	return regexp.MustCompile(`var\s+_badCredentialsCount\s*=\s*"0"`).MatchString(html) ||
		(strings.Contains(html, "getCaptcha.htl") && strings.Contains(html, "captchaDiv"))
}

func authServerEncryptPassword(password, salt string) (string, error) {
	key := []byte(strings.TrimSpace(salt))
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	iv := []byte(randomString(16))
	payload := []byte(randomString(64) + password)
	padding := aes.BlockSize - len(payload)%aes.BlockSize
	payload = append(payload, bytes.Repeat([]byte{byte(padding)}, padding)...)
	encrypted := make([]byte, len(payload))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(encrypted, payload)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func authServerBase(loginURL string) (string, error) {
	parsed, err := url.Parse(loginURL)
	if err != nil {
		return "", err
	}
	path := parsed.Path
	index := strings.Index(path, "/login")
	contextPath := "/authserver"
	if index >= 0 {
		contextPath = path[:index]
	}
	return parsed.Scheme + "://" + parsed.Host + contextPath, nil
}

func inputValue(html, name, elementID string) string {
	var pattern string
	if elementID != "" {
		pattern = `<input[^>]*id=["']` + regexp.QuoteMeta(elementID) + `["'][^>]*>`
	} else {
		pattern = `<input[^>]*name=["']` + regexp.QuoteMeta(name) + `["'][^>]*>`
	}
	input := regexp.MustCompile(`(?i)` + pattern).FindString(html)
	if input == "" {
		return ""
	}
	value := regexp.MustCompile(`(?i)value=["']([^"']*)`).FindStringSubmatch(input)
	if len(value) < 2 {
		return ""
	}
	return value[1]
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func randomString(length int) string {
	buffer := make([]byte, length)
	_, _ = rand.Read(buffer)
	builder := strings.Builder{}
	for _, item := range buffer {
		builder.WriteByte(aesChars[int(item)%len(aesChars)])
	}
	return builder.String()
}
