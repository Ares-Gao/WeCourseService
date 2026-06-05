package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type ClassTimeSlot struct {
	Start string
	End   string
}

type Config struct {
	SchoolName                string
	MangerType                string
	MangerURL                 string
	CalendarFirst             string
	SocketPort                int
	LoginType                 string
	AuthServerURL             string
	ServiceURL                string
	AuthServerAutoCaptcha     bool
	AuthServerCaptchaRetries  int
	DdddOcrOnnxRuntimeLibPath string
	DdddOcrModelPath          string
	DdddOcrDictPath           string
	DdddOcrDetModelPath       string
	DdddOcrUseCustomModel     bool
	CalendarTimezone          string
	CalendarName              string
	ClassTimeSlots            []ClassTimeSlot
}

type ConfigFile struct {
	Language string `json:"language"`
	Path     string `json:"path"`
	Exists   bool   `json:"exists"`
}

type State struct {
	Root   string       `json:"root"`
	Files  []ConfigFile `json:"files"`
	Config Config       `json:"config"`
}

func main() {
	addr := flag.String("addr", "127.0.0.1:9630", "ConfigTool listen address")
	openBrowser := flag.Bool("open", true, "open browser after start")
	flag.Parse()

	root, err := findRepoRoot()
	if err != nil {
		log.Fatal(err)
	}
	store := configStore{root: root}
	if _, err := store.state(); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(indexHTML))
	})
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		state, err := store.state()
		writeJSON(w, state, err)
	})
	mux.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var conf Config
		if err := json.NewDecoder(r.Body).Decode(&conf); err != nil {
			writeJSON(w, nil, err)
			return
		}
		state, err := store.save(normalizeConfig(conf))
		writeJSON(w, state, err)
	})
	mux.HandleFunc("/api/default", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, defaultConfig(), nil)
	})

	url := "http://" + *addr
	fmt.Printf("WeCourseService ConfigTool: %s\n", url)
	if *openBrowser {
		go func() {
			time.Sleep(300 * time.Millisecond)
			_ = openURL(url)
		}()
	}
	log.Fatal(http.ListenAndServe(*addr, mux))
}

type configStore struct {
	root string
}

func (s configStore) paths() []ConfigFile {
	rootConfig := filepath.Join(s.root, "config.json")
	goConfig := filepath.Join(s.root, "go", "config.json")
	return []ConfigFile{
		{Language: "Python / C# / PHP / Java", Path: rootConfig, Exists: fileExists(rootConfig)},
		{Language: "Go", Path: goConfig, Exists: fileExists(goConfig)},
	}
}

func (s configStore) state() (State, error) {
	conf, err := s.load()
	if err != nil {
		return State{}, err
	}
	if err := s.ensureFiles(conf); err != nil {
		return State{}, err
	}
	return State{Root: s.root, Files: s.paths(), Config: conf}, nil
}

func (s configStore) load() (Config, error) {
	for _, item := range s.paths() {
		if !item.Exists {
			continue
		}
		conf, err := readConfig(item.Path)
		if err == nil {
			return normalizeConfig(conf), nil
		}
	}
	return defaultConfig(), nil
}

func (s configStore) ensureFiles(conf Config) error {
	for _, item := range s.paths() {
		if item.Exists {
			continue
		}
		if err := writeConfig(item.Path, conf); err != nil {
			return err
		}
	}
	return nil
}

func (s configStore) save(conf Config) (State, error) {
	if err := s.writeAll(conf); err != nil {
		return State{}, err
	}
	return State{Root: s.root, Files: s.paths(), Config: conf}, nil
}

func (s configStore) writeAll(conf Config) error {
	for _, item := range s.paths() {
		if err := writeConfig(item.Path, conf); err != nil {
			return err
		}
	}
	return nil
}

func readConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, err
	}
	conf := defaultConfig()
	setString(raw, "SchoolName", &conf.SchoolName)
	setString(raw, "MangerType", &conf.MangerType)
	setString(raw, "MangerURL", &conf.MangerURL)
	setString(raw, "CalendarFirst", &conf.CalendarFirst)
	setInt(raw, "SocketPort", &conf.SocketPort)
	setString(raw, "LoginType", &conf.LoginType)
	setString(raw, "AuthServerURL", &conf.AuthServerURL)
	setString(raw, "ServiceURL", &conf.ServiceURL)
	setBool(raw, "AuthServerAutoCaptcha", &conf.AuthServerAutoCaptcha)
	setInt(raw, "AuthServerCaptchaRetries", &conf.AuthServerCaptchaRetries)
	setString(raw, "DdddOcrOnnxRuntimeLibPath", &conf.DdddOcrOnnxRuntimeLibPath)
	setString(raw, "DdddOcrModelPath", &conf.DdddOcrModelPath)
	setString(raw, "DdddOcrDictPath", &conf.DdddOcrDictPath)
	setString(raw, "DdddOcrDetModelPath", &conf.DdddOcrDetModelPath)
	setBool(raw, "DdddOcrUseCustomModel", &conf.DdddOcrUseCustomModel)
	setString(raw, "CalendarTimezone", &conf.CalendarTimezone)
	setString(raw, "CalendarName", &conf.CalendarName)
	if slots, ok := raw["ClassTimeSlots"].([]any); ok {
		conf.ClassTimeSlots = make([]ClassTimeSlot, 0, len(slots))
		for _, slot := range slots {
			item, ok := slot.(map[string]any)
			if !ok {
				continue
			}
			conf.ClassTimeSlots = append(conf.ClassTimeSlots, ClassTimeSlot{
				Start: stringValue(item["Start"]),
				End:   stringValue(item["End"]),
			})
		}
	}
	return conf, nil
}

func writeConfig(path string, conf Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(conf, "", "\t")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func normalizeConfig(conf Config) Config {
	defaults := defaultConfig()
	if strings.TrimSpace(conf.SchoolName) == "" {
		conf.SchoolName = defaults.SchoolName
	}
	if strings.TrimSpace(conf.MangerType) == "" {
		conf.MangerType = defaults.MangerType
	}
	if strings.TrimSpace(conf.MangerURL) == "" {
		conf.MangerURL = defaults.MangerURL
	}
	if strings.TrimSpace(conf.CalendarFirst) == "" {
		conf.CalendarFirst = defaults.CalendarFirst
	}
	if conf.SocketPort <= 0 {
		conf.SocketPort = defaults.SocketPort
	}
	if strings.TrimSpace(conf.LoginType) == "" {
		conf.LoginType = defaults.LoginType
	}
	if conf.AuthServerCaptchaRetries <= 0 {
		conf.AuthServerCaptchaRetries = defaults.AuthServerCaptchaRetries
	}
	if strings.TrimSpace(conf.CalendarTimezone) == "" {
		conf.CalendarTimezone = defaults.CalendarTimezone
	}
	if strings.TrimSpace(conf.CalendarName) == "" {
		conf.CalendarName = defaults.CalendarName
	}
	if len(conf.ClassTimeSlots) == 0 {
		conf.ClassTimeSlots = defaults.ClassTimeSlots
	}
	return conf
}

func defaultConfig() Config {
	return Config{
		SchoolName:               "山东商业职业技术学院",
		MangerType:               "supwisdom",
		MangerURL:                "http://szyjxgl.sict.edu.cn:9000/",
		CalendarFirst:            "2020-08-24",
		SocketPort:               25565,
		LoginType:                "direct",
		AuthServerAutoCaptcha:    true,
		AuthServerCaptchaRetries: 3,
		CalendarTimezone:         "Asia/Shanghai",
		CalendarName:             "微课表",
		ClassTimeSlots:           defaultClassTimeSlots(),
		DdddOcrUseCustomModel:    false,
	}
}

func defaultClassTimeSlots() []ClassTimeSlot {
	return []ClassTimeSlot{
		{Start: "08:00", End: "08:45"},
		{Start: "08:55", End: "09:40"},
		{Start: "10:00", End: "10:45"},
		{Start: "10:55", End: "11:40"},
		{Start: "14:00", End: "14:45"},
		{Start: "14:55", End: "15:40"},
		{Start: "16:00", End: "16:45"},
		{Start: "16:55", End: "17:40"},
		{Start: "19:00", End: "19:45"},
		{Start: "19:55", End: "20:40"},
		{Start: "20:50", End: "21:35"},
		{Start: "21:45", End: "22:30"},
	}
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(dir, "README.md")) && dirExists(filepath.Join(dir, "go")) {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return "", errors.New("cannot locate WeCourseService repo root")
}

func writeJSON(w http.ResponseWriter, value any, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}

func openURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func setString(raw map[string]any, key string, target *string) {
	if value := stringValue(raw[key]); value != "" {
		*target = value
	}
}

func setInt(raw map[string]any, key string, target *int) {
	switch value := raw[key].(type) {
	case float64:
		*target = int(value)
	case json.Number:
		if parsed, err := strconv.Atoi(value.String()); err == nil {
			*target = parsed
		}
	case string:
		if parsed, err := strconv.Atoi(value); err == nil {
			*target = parsed
		}
	}
}

func setBool(raw map[string]any, key string, target *bool) {
	switch value := raw[key].(type) {
	case bool:
		*target = value
	case string:
		if parsed, err := strconv.ParseBool(value); err == nil {
			*target = parsed
		}
	}
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>WeCourseService ConfigTool</title>
<style>
:root{color-scheme:light;--bg:#f7f8fa;--panel:#fff;--text:#20242a;--muted:#5e6673;--line:#d9dee7;--accent:#1d6f5f;--accent2:#b54b3a;--ok:#287a45}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--text);font:14px/1.45 system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}
header{display:flex;align-items:center;justify-content:space-between;gap:16px;padding:18px 28px;border-bottom:1px solid var(--line);background:#fff}
h1{margin:0;font-size:20px;font-weight:700}
main{max-width:1180px;margin:0 auto;padding:24px}
.toolbar{display:flex;align-items:center;gap:10px;flex-wrap:wrap}
button,select,input{font:inherit}
button{height:34px;border:1px solid var(--line);background:#fff;border-radius:6px;padding:0 12px;cursor:pointer}
button.primary{background:var(--accent);border-color:var(--accent);color:#fff}
button.danger{color:var(--accent2)}
.grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:18px}
section{background:var(--panel);border:1px solid var(--line);border-radius:8px;padding:18px}
section.wide{grid-column:1/-1}
h2{margin:0 0 14px;font-size:15px}
label{display:block;color:var(--muted);font-size:12px;margin:0 0 6px}
input,select{width:100%;height:36px;border:1px solid var(--line);border-radius:6px;background:#fff;color:var(--text);padding:6px 9px}
input[type=checkbox]{width:18px;height:18px}
.fields{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}
.check{display:flex;align-items:center;gap:9px;margin-top:24px;color:var(--text)}
.files{display:grid;gap:8px}
.file{display:grid;grid-template-columns:180px 1fr 80px;gap:10px;align-items:center;border:1px solid var(--line);border-radius:6px;padding:9px;background:#fbfcfd}
.status{font-weight:600;color:var(--ok)}
.status.missing{color:var(--accent2)}
table{width:100%;border-collapse:collapse}
th,td{border-bottom:1px solid var(--line);padding:8px;text-align:left}
th{font-size:12px;color:var(--muted);font-weight:600}
td:first-child{width:70px;color:var(--muted)}
.slot-actions{display:flex;gap:8px;margin-top:12px}
#message{min-height:20px;color:var(--muted)}
@media (max-width:760px){main{padding:16px}.grid,.fields{grid-template-columns:1fr}.file{grid-template-columns:1fr}header{align-items:flex-start;flex-direction:column}}
</style>
</head>
<body>
<header>
  <div>
    <h1>WeCourseService ConfigTool</h1>
    <div id="message"></div>
  </div>
  <div class="toolbar">
    <select id="lang" aria-label="Language"><option value="zh">中文</option><option value="en">English</option></select>
    <button id="reload"></button>
    <button id="defaults"></button>
    <button id="save" class="primary"></button>
  </div>
</header>
<main>
  <div class="grid">
    <section class="wide">
      <h2 data-i18n="files"></h2>
      <div class="files" id="files"></div>
    </section>
    <section>
      <h2 data-i18n="basic"></h2>
      <div class="fields">
        <div><label data-i18n="SchoolName"></label><input id="SchoolName"></div>
        <div><label data-i18n="MangerType"></label><input id="MangerType"></div>
        <div><label data-i18n="MangerURL"></label><input id="MangerURL"></div>
        <div><label data-i18n="CalendarFirst"></label><input id="CalendarFirst" type="date"></div>
        <div><label data-i18n="SocketPort"></label><input id="SocketPort" type="number" min="1" max="65535"></div>
        <div><label data-i18n="LoginType"></label><select id="LoginType"><option value="direct">direct</option><option value="authserver">authserver</option></select></div>
      </div>
    </section>
    <section>
      <h2 data-i18n="auth"></h2>
      <div class="fields">
        <div><label data-i18n="AuthServerURL"></label><input id="AuthServerURL"></div>
        <div><label data-i18n="ServiceURL"></label><input id="ServiceURL"></div>
        <div><label data-i18n="AuthServerCaptchaRetries"></label><input id="AuthServerCaptchaRetries" type="number" min="1" max="20"></div>
        <label class="check"><input id="AuthServerAutoCaptcha" type="checkbox"><span data-i18n="AuthServerAutoCaptcha"></span></label>
      </div>
    </section>
    <section>
      <h2 data-i18n="ocr"></h2>
      <div class="fields">
        <div><label data-i18n="DdddOcrOnnxRuntimeLibPath"></label><input id="DdddOcrOnnxRuntimeLibPath"></div>
        <div><label data-i18n="DdddOcrModelPath"></label><input id="DdddOcrModelPath"></div>
        <div><label data-i18n="DdddOcrDictPath"></label><input id="DdddOcrDictPath"></div>
        <div><label data-i18n="DdddOcrDetModelPath"></label><input id="DdddOcrDetModelPath"></div>
        <label class="check"><input id="DdddOcrUseCustomModel" type="checkbox"><span data-i18n="DdddOcrUseCustomModel"></span></label>
      </div>
    </section>
    <section>
      <h2 data-i18n="calendar"></h2>
      <div class="fields">
        <div><label data-i18n="CalendarTimezone"></label><input id="CalendarTimezone"></div>
        <div><label data-i18n="CalendarName"></label><input id="CalendarName"></div>
      </div>
    </section>
    <section class="wide">
      <h2 data-i18n="slots"></h2>
      <table>
        <thead><tr><th data-i18n="slot"></th><th data-i18n="start"></th><th data-i18n="end"></th><th></th></tr></thead>
        <tbody id="slotRows"></tbody>
      </table>
      <div class="slot-actions">
        <button id="addSlot"></button>
      </div>
    </section>
  </div>
</main>
<script>
const text={
 zh:{files:"配置文件",basic:"基础配置",auth:"统一认证",ocr:"Go 验证码识别",calendar:"ICS 日历",slots:"上课时间表",slot:"节次",start:"开始",end:"结束",reload:"重新读取",defaults:"默认值",save:"保存全部",addSlot:"新增节次",remove:"删除",exists:"存在",missing:"已自动生成",saved:"已保存",loaded:"已读取",defaulted:"已载入默认值",SchoolName:"学校名称",MangerType:"教务系统类型",MangerURL:"教务系统地址",CalendarFirst:"第一周周一",SocketPort:"WebSocket 端口",LoginType:"默认登录方式",AuthServerURL:"AuthServer 登录地址",ServiceURL:"SSO Service 地址",AuthServerAutoCaptcha:"自动识别验证码",AuthServerCaptchaRetries:"验证码重试次数",DdddOcrOnnxRuntimeLibPath:"onnxruntime 动态库路径",DdddOcrModelPath:"ddddocr 模型路径",DdddOcrDictPath:"ddddocr 字典路径",DdddOcrDetModelPath:"ddddocr 检测模型路径",DdddOcrUseCustomModel:"使用自定义模型",CalendarTimezone:"日历时区",CalendarName:"日历名称"},
 en:{files:"Config files",basic:"Basic",auth:"Authserver",ocr:"Go captcha OCR",calendar:"ICS calendar",slots:"Class time slots",slot:"No.",start:"Start",end:"End",reload:"Reload",defaults:"Defaults",save:"Save all",addSlot:"Add slot",remove:"Remove",exists:"Exists",missing:"Generated",saved:"Saved",loaded:"Loaded",defaulted:"Defaults loaded",SchoolName:"School name",MangerType:"Manager type",MangerURL:"Manager URL",CalendarFirst:"First Monday",SocketPort:"WebSocket port",LoginType:"Default login type",AuthServerURL:"Authserver login URL",ServiceURL:"SSO service URL",AuthServerAutoCaptcha:"Auto solve captcha",AuthServerCaptchaRetries:"Captcha retries",DdddOcrOnnxRuntimeLibPath:"onnxruntime library path",DdddOcrModelPath:"ddddocr model path",DdddOcrDictPath:"ddddocr dict path",DdddOcrDetModelPath:"ddddocr detection model path",DdddOcrUseCustomModel:"Use custom model",CalendarTimezone:"Calendar timezone",CalendarName:"Calendar name"}
};
const fields=["SchoolName","MangerType","MangerURL","CalendarFirst","SocketPort","LoginType","AuthServerURL","ServiceURL","AuthServerAutoCaptcha","AuthServerCaptchaRetries","DdddOcrOnnxRuntimeLibPath","DdddOcrModelPath","DdddOcrDictPath","DdddOcrDetModelPath","DdddOcrUseCustomModel","CalendarTimezone","CalendarName"];
let config={ClassTimeSlots:[]};
let lang=localStorage.getItem("wecourse-configtool-lang")||"zh";
document.getElementById("lang").value=lang;
function t(k){return text[lang][k]||k}
function applyLang(){document.querySelectorAll("[data-i18n]").forEach(el=>el.textContent=t(el.dataset.i18n));["reload","defaults","save","addSlot"].forEach(id=>document.getElementById(id).textContent=t(id));renderSlots();renderFiles(window.lastFiles||[])}
function setMessage(key){document.getElementById("message").textContent=t(key)}
function fillForm(){fields.forEach(id=>{const el=document.getElementById(id);if(!el)return;if(el.type==="checkbox")el.checked=!!config[id];else el.value=config[id]??""});renderSlots()}
function readForm(){fields.forEach(id=>{const el=document.getElementById(id);if(el.type==="checkbox")config[id]=el.checked;else if(el.type==="number")config[id]=Number(el.value||0);else config[id]=el.value});config.ClassTimeSlots=[...document.querySelectorAll("#slotRows tr")].map(row=>({Start:row.querySelector(".start").value,End:row.querySelector(".end").value})).filter(x=>x.Start||x.End);return config}
function renderFiles(files){window.lastFiles=files;const box=document.getElementById("files");box.innerHTML="";files.forEach(file=>{const row=document.createElement("div");row.className="file";row.innerHTML="<strong>"+file.language+"</strong><span>"+file.path+"</span><span class=\"status "+(file.exists?"":"missing")+"\">"+(file.exists?t("exists"):t("missing"))+"</span>";box.appendChild(row)})}
function renderSlots(){const body=document.getElementById("slotRows");if(!body)return;body.innerHTML="";(config.ClassTimeSlots||[]).forEach((slot,i)=>{const row=document.createElement("tr");row.innerHTML="<td>"+(i+1)+"</td><td><input class=\"start\" type=\"time\" value=\""+(slot.Start||"")+"\"></td><td><input class=\"end\" type=\"time\" value=\""+(slot.End||"")+"\"></td><td><button class=\"danger\" type=\"button\">"+t("remove")+"</button></td>";row.querySelector("button").onclick=()=>{config.ClassTimeSlots.splice(i,1);renderSlots()};body.appendChild(row)})}
async function loadState(){const res=await fetch("/api/state");const data=await res.json();if(data.error)throw new Error(data.error);config=data.config;renderFiles(data.files);fillForm();setMessage("loaded")}
async function loadDefaults(){const res=await fetch("/api/default");config=await res.json();fillForm();setMessage("defaulted")}
async function save(){readForm();const res=await fetch("/api/save",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(config)});const data=await res.json();if(data.error)throw new Error(data.error);config=data.config;renderFiles(data.files);fillForm();setMessage("saved")}
document.getElementById("lang").onchange=e=>{lang=e.target.value;localStorage.setItem("wecourse-configtool-lang",lang);applyLang()};
document.getElementById("reload").onclick=()=>loadState().catch(err=>setMessage(err.message));
document.getElementById("defaults").onclick=()=>loadDefaults().catch(err=>setMessage(err.message));
document.getElementById("save").onclick=()=>save().catch(err=>setMessage(err.message));
document.getElementById("addSlot").onclick=()=>{readForm();config.ClassTimeSlots.push({Start:"",End:""});renderSlots()};
applyLang();loadState().catch(err=>setMessage(err.message));
</script>
</body>
</html>`
