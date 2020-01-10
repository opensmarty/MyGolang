/*
  1. 开发一个 http 接口，用于接收数据
  2. 将 接收到的 http数据 以文件的形式上传至 ftp 服务器
  注意：只需要一个ftp账号即可，安全，方便
  http 监听端口：5656
  ftp Addr：192.168.160.8:21
  ftp 账号： yang.yang
  ftp 密码：12345678
*/
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	//"strings"

	//"io"
	"log"
	"net/http"

	"os"
	"time"

	"github.com/jlaffaye/ftp"
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
	workDir = "/dataTask"
)

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
	myHttp     *http.Server
	ftpClient  *ftp.ServerConn
	policeData chan *CallPolice
}

type program struct{}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
	// 代码写在这儿
	myServer := &Server{}
	myServer.policeData = make(chan *CallPolice, 128)
	myServer.myHttp = new(http.Server)
	mux := http.NewServeMux()
	mux.HandleFunc("/Location/callPolice", myServer.callPolice)
	mux.HandleFunc("/Location/hangUp", myServer.hangUp)
	mux.HandleFunc("/Location/policeStation", myServer.policeStation)
	myServer.myHttp.Handler = mux

	ftpClient, err := ftp.DialTimeout("192.168.160.8:21", 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	err = ftpClient.Login("yang.yang", "12345678")
	if err != nil {
		log.Fatal(err)
	}
	/*
		err = ftpClient.MakeDir("dataTask")
		if err != nil {
			if strings.Contains(err.Error(), "Directory already exists") {
				log.Println("Directory already exists")
			}
		}
	*/
	currentDir, err := ftpClient.CurrentDir()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(currentDir)
	if currentDir != workDir {
		err = ftpClient.ChangeDir("dataTask")
		if err != nil {
			log.Fatal(err)
		}
	}
	currentDir, err = ftpClient.CurrentDir()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(currentDir)
	myServer.ftpClient = ftpClient
	//path := "/gonggong/abc.txt"
	go myServer.handleCallPolice()
	myServer.myHttp.Addr = ":5656"
	herr := myServer.myHttp.ListenAndServe()
	if herr != nil {
		log.Fatal(herr)
	}
	select {}
}

func (p *program) Stop(s service.Service) error {
	return nil
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
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	rCallPolice := &CallPolice{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(rCallPolice)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	fmt.Println("fffff", rCallPolice)
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
			fmt.Println("77777777777")
			caller := msg.Bjhm
			fmt.Println(caller)
			fileName := caller + "_" + strconv.Itoa(int(time.Now().Unix()))

			sMsg, err := json.Marshal(msg)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(caller, fileName, len(sMsg))
			data := bytes.NewBufferString(string(sMsg))

			//sbuf := &bytes.Buffer{}
			//sbuf.Write(sMsg)
			//data := bytes.NewBuffer(sMsg)

			err = s.ftpClient.Stor(fileName, data)
			if err != nil {
				fmt.Println(err.Error())
				if err = s.ftpClient.Quit(); err != nil {
					log.Fatal(err)
				}
				panic(err)
			}
		}
	}
	return
}
func (s *Server) hangUp(w http.ResponseWriter, r *http.Request)        {}
func (s *Server) policeStation(w http.ResponseWriter, r *http.Request) {}
