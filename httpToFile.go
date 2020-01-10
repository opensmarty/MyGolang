/*
	1. 开发一个 http 接口，用于接收数据（urlencode）
  	2. 将 接收到的 http数据 以文件的形式存储至本地目录
	3. 作为 windows 服务跑起来
*/
package main

import (
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strings"

	//"bytes"
	"encoding/json"
	"strconv"

	//"strings"
	"log"
	"net/http"

	"os"
	"time"

	"github.com/jander/golog/logger"
	//"github.com/jlaffaye/ftp"
	"github.com/kardianos/service"
)

/*
{
  “area_code”:”区域编号”,
  “jkxh”:”本接口序号，这里填2”,
 “bjhm”:”拨打110的报警号码”,
“sthy_lsh”:”三台合一临时组群流水号”,
  “zxh”:”三台合一110坐席号”,
  “fscs”:”这个接口发送的次数，如：1或2”,
  “qhfw”:”需要圈呼的范围（单位米）”,
  “lng”:”经度（国标）”,
  “lat”:”纬度（国标）”
}
*/
const (
	workDir     = "D:\\FTP\\YDHDJ1"
	logFileName = "Rabbit.log"
	httpPort    = ":5656"
)

var rotatingHandler *logger.RotatingHandler

type CallPolice struct {
	AreaCode string `json:"area_code"`
	Jkxh     string `json:"jkxh"`
	Bjhm     string `json:"bjhm"`
	SthyLsh  string `json:"sthy_lsh"`
	Zxh      string `json:"zxh"`
	Fscs     string `json:"fscs"`
	Qhfw     string `json:"qhfw"`
	Lng      string `json:"lng"`
	Lat      string `json:"lat"`
}

type Server struct {
	myHttp *http.Server
	//ftpClient  *ftp.ServerConn
	policeData chan *CallPolice
}

type program struct{}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
	// 代码写在这儿
	dir, _ := os.Executable()
	exPath := filepath.Dir(dir)
	sstr := strings.Split(exPath, "\\bin")

	logDir := sstr[0] + "\\log"
	log.Println("logDir:", logDir)
	if !checkPathExist(logDir) {
		//logger.Error("log folder not exist!")
		os.Mkdir(logDir, 0664)
	}
	rotatingHandler = logger.NewRotatingHandler(logDir, logFileName, 5, 20*1024*1024)
	logger.SetHandlers(logger.Console, rotatingHandler)
	defer logger.Close()
	logger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logger.SetLevel(logger.INFO)

	myServer := &Server{}
	myServer.policeData = make(chan *CallPolice, 128)
	myServer.myHttp = new(http.Server)
	mux := http.NewServeMux()
	mux.HandleFunc("/Location/callPolice", myServer.callPolice)
	mux.HandleFunc("/Location/hangUp", myServer.hangUp)
	mux.HandleFunc("/Location/policeStation", myServer.policeStation)
	myServer.myHttp.Handler = mux

	if !checkPathExist(workDir) {
		rotatingHandler.Error("workDir not exist!", "Exit!")
		os.Exit(1)
	}
	go myServer.handleCallPolice()
	myServer.myHttp.Addr = httpPort
	herr := myServer.myHttp.ListenAndServe()
	if herr != nil {
		/*注意：直接用 logger.Error() ，会导致定位错误的代码文件和行号*/
		rotatingHandler.Error("Init httpServer err:", herr.Error())
		os.Exit(1)
	}
	rotatingHandler.Info("===== start =====")
	select {}
}

func (p *program) Stop(s service.Service) error {
	return nil
}
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
func checkPathExist(dir string) bool {
	var exist = true
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

/**
 * MAIN函数，程序入口
 */
func main() {
	svcConfig := &service.Config{
		Name:        "rabbitShuguo", //服务显示名称
		DisplayName: "rabbitShuguo", //服务名称
		Description: "rabbitShuguo", //服务描述
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 {
		if os.Args[1] == "install" {
			s.Install()
			log.Println("服务安装成功")
			return
		}

		if os.Args[1] == "remove" {
			s.Uninstall()
			log.Println("服务卸载成功")
			return
		}
	}

	err = s.Run()
	if err != nil {
		log.Fatal(err)
	}
}
func (s *Server) callPolice(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	rCallPolice := &CallPolice{}
	rBody, rerr := ioutil.ReadAll(r.Body)
	if rerr != nil {
		rotatingHandler.Error("rBody err:", r.Body)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//decoder := json.NewDecoder(r.Body)
	//err := decoder.Decode(rCallPolice)

	rBodyStr, _ := url.QueryUnescape(string(rBody))
	err := json.Unmarshal([]byte(rBodyStr), rCallPolice)
	if err != nil {
		rotatingHandler.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rotatingHandler.Info("RecvHttpData:", *rCallPolice)
	s.policeData <- rCallPolice
	// client 用ajax请求
	w.Header().Add("Access-Control-Allow-Origin", "*")
	//w.Write(ret)
	w.WriteHeader(200)
}
func (s *Server) handleCallPolice() {
	for {
		select {
		case msg := <-s.policeData:
			caller := msg.Bjhm
			fileName := caller + "_" + strconv.Itoa(int(time.Now().Unix()))
			dataFileName := workDir + "\\" + fileName
			sMsg, err := json.Marshal(msg)
			if err != nil {
				rotatingHandler.Error(err.Error())
			}
			rotatingHandler.Info(caller, dataFileName, len(sMsg))
			fHandle, err := os.Create(dataFileName)
			if err != nil {
				rotatingHandler.Info("Create file err:", err.Error())
				if fHandle != nil {
					fHandle.Close()
				}
				continue
			}
			_, err = fHandle.Write(sMsg)
			if err != nil {
				rotatingHandler.Error("Write file err:", err.Error())
			}

			rotatingHandler.Info("Write file OK!")
			if fHandle != nil {
				fHandle.Close()
			}
		}
	}
	return
}
func (s *Server) hangUp(w http.ResponseWriter, r *http.Request)        {}
func (s *Server) policeStation(w http.ResponseWriter, r *http.Request) {}
