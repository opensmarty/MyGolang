/*
 * 背景：第三方程序不断向 windows 某个目录存放数据文件（假设目录名叫 files）
 * 功能：监听 windows 上的 files 目录，消费数据文件，消费完成后将其删除
 * 消费数据文件：读取文件内容，并将内容发向指定接口
 *
 * 坑1：获取执行程序的相对路径
 *      因为要保证程序放到哪里都可以运行，所以指定配置文件，log文件时都要用相对路径
 *      获取 相对路径的方式有两种：os.Getwd() 和 dir, _ := os.Executable();exPath := filepath.Dir(dir) 第一种方式某些时候不好用
 *
 * 坑2：此程序是在linux平台书写编译，跑在 windows上，在 windows上，程序要注册成服务在后台运行(用nc工具)，所以要使用 windows 框架，把程序封装起来
 *      "github.com/kardianos/service"
 * 坑3：windows 平台的目录引用
 *      错误的引用方式："D:\FTP\YDHDJ1" 或 "D:/FTP/YDHDJ1"
 *      正确的引用方式："D:\\FTP\\YDHDJ1"
 * 编译：GOOS=windows GOARCH=amd64 go build folderMoniter.go
 */
 

package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jander/golog/logger"
	"github.com/kardianos/service"
)

const (
	watchdir    = "D:\\FTP\\YDHDJ1"
	platformAPI = "http://36.189.252.169:48000/SG_sns/index.php?app=api&mod=SharXj&act=parseShortNumber"
	logFileName = "folderMoniter.log"
)

var PoliceMsg chan *CallPolice
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

	rotatingHandler = logger.NewRotatingHandler(logDir, logFileName, 5, 10*1024*1024)
	logger.SetHandlers(logger.Console, rotatingHandler)
	defer logger.Close()
	logger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logger.SetLevel(logger.INFO)

	PoliceMsg = make(chan *CallPolice, 10)
	var err error
	if !checkPathExist(watchdir) {
		rotatingHandler.Error("watchdir not exist:", watchdir)
		os.Exit(1)
	}
	go sendMsg()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		rotatingHandler.Error(err)
		os.Exit(1)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					rotatingHandler.Error("Event error!")
					close(done)
					return
				}
				switch event.Op {
				case fsnotify.Create:
					rotatingHandler.Info("create:", event.Name)
					if strings.HasPrefix(event.Name, ".") {
						rotatingHandler.Info("file type err ")
						continue
					}
					fullFileName := event.Name
					time.Sleep(100 * time.Millisecond)
					byteData := getFileData(fullFileName)
					if len(byteData) == 0 {
						continue
					}
					rotatingHandler.Info("get file content:", string(byteData))
					err = os.Remove(fullFileName)
					if err != nil {
						rotatingHandler.Info("Remove:", err.Error())
					}

					rdata := &CallPolice{}
					jerr := json.Unmarshal(byteData, rdata)
					if jerr != nil {
						rotatingHandler.Info(jerr.Error())
						continue
					}
					PoliceMsg <- rdata

				case fsnotify.Rename, fsnotify.Remove:
					rotatingHandler.Info("remove:", event.Name)
				case fsnotify.Write:
					rotatingHandler.Info("write:", event.Name)
				}
			case err := <-watcher.Errors:
				rotatingHandler.Info("error:", err)
			}
		}
	}()
	err = watcher.Add(watchdir)
	if err != nil {
		rotatingHandler.Error(err)
		os.Exit(1)
	}
	<-done
}
func (p *program) Stop(s service.Service) error {
	return nil
}
func main() {
	svcConfig := &service.Config{
		Name:        "FolderMoniter", //服务显示名称
		DisplayName: "FolderMoniter", //服务名称
		Description: "FolderMoniter", //服务描述
	}
	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		rotatingHandler.Fatal(err)
	}
	if err != nil {
		rotatingHandler.Fatal(err)
	}
	if len(os.Args) > 1 {
		if os.Args[1] == "install" {
			s.Install()
			rotatingHandler.Println("服务安装成功")
			return
		}
		if os.Args[1] == "remove" {
			s.Uninstall()
			rotatingHandler.Println("服务卸载成功")
			return
		}
	}
	err = s.Run()
	if err != nil {
		rotatingHandler.Error(err)
	}
}

func getFileData(fileName string) []byte {

	fileHandler, err := os.Open(fileName)
	defer func() {
		if fileHandler != nil {
			fileHandler.Close()
		}
	}()
	if err != nil {
		rotatingHandler.Error("Open file err:", err.Error())
		return nil
	}
	fileBuf := make([]byte, 0)
	n, err := fileHandler.Read(fileBuf)
	rotatingHandler.Info(n)
	if err != nil {
		rotatingHandler.Error("Open file err:", err.Error())
	}

	return fileBuf[:n]
}
func checkPathExist(dir string) bool {
	var exist = true
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func sendMsg() {
	for {
		rdata, ok := <-PoliceMsg
		if !ok {
			return
		}
		data := make(url.Values)
		data["area_code"] = []string{rdata.AreaCode}
		data["jkxh"] = []string{rdata.Jkxh}
		data["bjhm"] = []string{rdata.Bjhm}
		data["sthy_lsh"] = []string{rdata.SthyLsh}
		data["zxh"] = []string{rdata.Zxh}
		data["fscs"] = []string{rdata.Fscs}
		data["qhfw"] = []string{rdata.Qhfw}
		data["lng"] = []string{rdata.Lng}
		data["lat"] = []string{rdata.Lat}
		go func() {
			client := &http.Client{}
			presp, perr := client.PostForm(platformAPI, data)
			if perr != nil {
				rotatingHandler.Error(perr.Error())
				return
			}

			rbody, err := ioutil.ReadAll(presp.Body)
			if err != nil {
				rotatingHandler.Error(err.Error())
			}
			rotatingHandler.Info(string(rbody))
			presp.Body.Close()
			return
		}()
	}
}

func substr(s string, pos, length int) string {
	runes := []rune(s)
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[pos:l])
}
