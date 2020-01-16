/*

部署：
	1. 解压 cat.zip 至任意目录
	2. 进入 cat/bin/
	3. 启动："以管理员身份运行" start.bat
	4. 停止："以管理员身份运行" stop.bat
  功能：
	
	1. 加载 task 文件 Cat\task\Book.xlsx 
	2. 向人员的邮箱发送对应的信息
	3. log文件
	4. 开机自启动
	
检查：
	任务管理器进程名：Cat
	
文件说明：
	主程序			Cat\bin\cat.exe (64位系统上运行)
	启动程序		Cat\bin\start.bat 
	停止程序		Cat\bin\stop.bat
	配置文件		Cat\conf\Cat.conf (json 格式)
					{
						"eMailUserName":"sw.dev@chinashuguo.com",	// 发送者邮箱地址
						"eMailUserPwd":"xxxxxx",				// 发送者邮箱密码
						"excelFileName":"Book.xlsx"					// task文件名
					}
	log文件			Cat\log\Cat.log
	task文件		Cat\task\Book.xlsx
	
注意：
	如果开了360软件请将加入可信

*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/go-gomail/gomail"
	"github.com/jander/golog/logger"
	"github.com/kardianos/service"
	"github.com/tealeg/xlsx"
)

var (
	myGmailDialer *gomail.Dialer
	//emailSuffix     string = "@chinashuguo.com"
	rotatingHandler *logger.RotatingHandler
	MyConf          = &myConf{}
	logFileName     = "Cat.log"
	//excelFileName   = "Book.xlsx"
	configFileName = "Cat.conf"
	eMailSubject   string
	sendOkCount    int32
	sendFalseCount int32
)

type userMsg struct {
	Id                 string `xlsx:"0"`
	MsgId              string `xlsx:"1"`
	UserName           string `xlsx:"2"`
	CompanyName        string `xlsx:"3"`
	Department         string `xlsx:"4"`
	HireDate           string `xlsx:"5"`
	EmailAddr          string `xlsx:"6"`
	PayDays            string `xlsx:"7"`
	TravelDays         string `xlsx:"8"`
	OvertimeDays       string `xlsx:"9"`
	PayForWorkDays     string `xlsx:"10"`
	PayForOvertimeDays string `xlsx:"11"`
	PayForTraffic      string `xlsx:"12"`
	PayForPhone        string `xlsx:"13"`
	PaySum             string `xlsx:"14"`
}

type myConf struct {
	MailUserName  string `json:"eMailUserName"`
	MailUserPwd   string `json:"eMailUserPwd"`
	ExcelFileName string `json:"excelFileName"`
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
	//log.Println("logDir:", logDir)
	if !checkPathExist(logDir) {
		os.Mkdir(logDir, 0664)
	}
	taskDir := sstr[0] + "\\task"
	//log.Println("logDir:", logDir)
	if !checkPathExist(taskDir) {
		os.Mkdir(taskDir, 0664)
	}

	rotatingHandler = logger.NewRotatingHandler(logDir, logFileName, 5, 10*1024*1024)
	logger.SetHandlers(logger.Console, rotatingHandler)
	defer logger.Close()
	logger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logger.SetLevel(logger.INFO)
	confDir := sstr[0] + "\\conf"
	//log.Println("logDir:", logDir)
	if !checkPathExist(confDir) {
		rotatingHandler.Fatal("config dir not exist!")
	}
	configFileFullName := confDir + "\\" + configFileName
	file, cerr := os.Open(configFileFullName)
	if cerr != nil {
		rotatingHandler.Fatal(cerr.Error())
	}
	defer file.Close()
	//syscall.Umask(0)
	decoder := json.NewDecoder(file)
	err := decoder.Decode(MyConf)
	if err != nil {
		rotatingHandler.Fatal("Error:", err)
	}
	rotatingHandler.Info("---------------- Cat start -------------")
	//rotatingHandler.Info("参数详情:")
	//rotatingHandler.Info(*MyConf)
	taskFileFullName := taskDir + "\\" + MyConf.ExcelFileName
	xlFile, err := xlsx.OpenFile(taskFileFullName)
	if err != nil {
		rotatingHandler.Fatal(err.Error())
	}
	titleMsg := &userMsg{}
	/* sheet: 表单
	* rows : 行
	* cells : 从左到右
	 */
	for _, sheet := range xlFile.Sheets {
		for rowIndex, row := range sheet.Rows {
			rotatingHandler.Info("----------- RowIndex ---------------: ", rowIndex)
			if rowIndex == 0 {
				for cellIndex, cell := range row.Cells {
					//rotatingHandler.Info("-------- cellIndex ------------: ", cellIndex)
					text := cell.String()
					if text != "" {
						rotatingHandler.Info("Subject: ", cellIndex, text)
						eMailSubject = text
					}
				}
				continue
			}
			if rowIndex == 1 {
				terr := row.ReadStruct(titleMsg)
				if terr != nil {
					rotatingHandler.Fatal(terr.Error())
				}
				continue
			}
			if rowIndex > 1 {
				m := gomail.NewMessage()
				tMsg := &userMsg{}
				terr := row.ReadStruct(tMsg)
				if terr != nil {
					rotatingHandler.Error(terr.Error())
					continue
				}
				sendMsg := fmt.Sprintf("%s: %s\r\n%s: %s\r\n%s: %s\r\n%s: %s\r\n%s: %s\r\n%s: %s\r\n%s: %s\r\n%s: %s\r\n%s: %s\r\n%s: %s\r\n%s: %s\r\n%s: %s\r\n", titleMsg.UserName, tMsg.UserName, titleMsg.CompanyName, tMsg.CompanyName, titleMsg.Department, tMsg.Department, titleMsg.HireDate, tMsg.HireDate, titleMsg.PayDays, tMsg.PayDays, titleMsg.TravelDays, tMsg.TravelDays, titleMsg.OvertimeDays, tMsg.OvertimeDays, titleMsg.PayForWorkDays, tMsg.PayForWorkDays, titleMsg.PayForOvertimeDays, tMsg.PayForOvertimeDays, titleMsg.PayForTraffic, tMsg.PayForTraffic, titleMsg.PayForPhone, tMsg.PayForPhone, titleMsg.PaySum, tMsg.PaySum)
				rotatingHandler.Info(sendMsg)
				if tMsg.EmailAddr == "" || eMailSubject == "" || MyConf.MailUserName == "" || sendMsg == "" {
					continue
				}
				m.SetHeader("From", MyConf.MailUserName)
				m.SetHeader("Subject", eMailSubject)
				m.SetHeader("To", tMsg.EmailAddr)
				/* 使用 "text/html" 类型，换行符不生效  */
				m.SetBody("text/plain", sendMsg)
				//m.SetBody("text/html", sendMsg)
				/*
					if *eAttachPath != "" {
						m.Attach(*eAttachPath)
					}
				*/
				go sendMsgToEmail(m, rowIndex)
				/*
					for cellIndex, cell := range row.Cells {
						//fmt.Printf("cellIndex: %d\t", cellIndex)
						text := cell.String()
						fmt.Printf("%20s\n", text)
					}
				*/
			}
		}
	}
	rotatingHandler.Info("********** stop ***** sendOkCount: ", sendOkCount, " ******* sendFalseCount: ", sendFalseCount, " *********")
}
func (p *program) Stop(s service.Service) error {
	return nil
}
func main() {
	svcConfig := &service.Config{
		Name:        "Cat", //服务显示名称
		DisplayName: "Cat", //服务名称
		Description: "Cat", //服务描述
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

func sendMsgToEmail(m *gomail.Message, RowIndex int) {
	//d := gomail.NewDialer("smtp.exmail.qq.com", 465, "sw.dev@chinashuguo.com", "!ShuGuo@123#")
	d := gomail.NewDialer("smtp.exmail.qq.com", 465, MyConf.MailUserName, MyConf.MailUserPwd)
	if err := d.DialAndSend(m); err != nil {
		rotatingHandler.Error(err.Error())
		rotatingHandler.Info("--------- RowIndex ", RowIndex, ": ", m.GetHeader("To"), "  Send False", "--------------")
		sendFalseCount = atomic.AddInt32(&sendFalseCount, 1)
		return
	}
	rotatingHandler.Info("--------- RowIndex ", RowIndex, ": ", m.GetHeader("To"), "  Send OK", "--------------")
	sendOkCount = atomic.AddInt32(&sendOkCount, 1)
	return
}
func checkPathExist(dir string) bool {
	var exist = true
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		exist = false
	}
	return exist
}
