/*
	背景： 第三方程序不断向 ftp用户的alarms目录生产数据，内容是json，有四种文件格式，我们只处理txt文件
			我方程序 消费该数据：读取文件内容（json格式），并将文件内容发送到指定接口
	功能：
			1. 代碼中使用相對路徑，程序放到任意目錄都可以運行
			2. 程序论寻遍历ftp目标目录，获取文件名
			3. 读取文件内容，并将内容发送到目标接口
			4. 删除处理过的txt文件，删除 十分钟之前的文件（有些问题文件读不出内容）
			5. 重连
	ftp库："github.com/jlaffaye/ftp"
	log库："github.com/jander/golog/logger"
	目錄結構：myPro/bin
				myPro/log
				myPro/conf
  坑1：
      "github.com/jlaffaye/ftp" github上 demo 中读文件的代码有误，直接copy使用会出大问题！
      {
      r, err := ftpClient.Retr(fullFileName) 
      …… ……
      buf, err := ioutil.ReadAll(r)
      …… ……
      }
       这个 r 在 函数返回前，必须 close 掉，否则会出大问题。代码的文档中有说明
   坑2：
      	for {
		        select {
		          case msg := <-msgChan:
			            if !msg {
			              	keepAliveChan <- msg
				              continue
			            }
		          case <-keepAliveChan:
		                	handleReconnect()
		        }
     	  }
      起不到重连的效果，keepAliveChan 不是每次都能执行到 ： 在 select {} 中使用 continue , return , goto 或 sleep 时，一定要注意其前面没有向后面的channel生产数据
   坑3：
       经观察发现 ftp句柄 读文件 或 拉文件名出错，需要15分钟才能返回
        原因暂时未知，需要跟一下代码
   坑4：向接口发送数据时，接口总是返回解析失败，原来是 http 的 "Content-Type" 不对。接口那边将 body以 json 字符串解析 对应的类型是 "application/json"
   	 而我们的头里填的是 表单 类型，body 却是 json 字符串类型，所以接口那边apache解析出错 client.Post(platformAPI, "application/x-www-form-urlencoded", body)
  心得：      
  不要浮躁，有问题了，要简化问题，挨个调试功能点，找出问题所在，然后潜心看代码，积极解决问题
  问题实在解决不了，再曲线救国，寻找替代方案
*/
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jander/golog/logger"
	"github.com/jlaffaye/ftp"
)

const (
	workDir     = "/alarms"
	platformAPI = "http://10.242.225.72:48000/SG_sns/index.php?app=api&mod=SharXj&act=parseFileData"
	logFileName = "passLocation.log"
	ftpAddr     = "117.146.60.17:8808"
	ftpUser     = "xxx"
	//ftpPwd 不用登錄密碼，ftp服務端設置了acl
)

var PoliceMsg chan []byte
var rotatingHandler *logger.RotatingHandler
var ftpClient *ftp.ServerConn
var ftpClientfordel *ftp.ServerConn
var keepAliveChann chan bool
var delFileChan chan string

func main() {
	/*取消系統對進程文件句柄數的限制*/
	var rlim syscall.Rlimit
	rlim.Cur = 1000000
	rlim.Max = 1000000
	err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)
	if err != nil {
		fmt.Println("set rlimit error: " + err.Error())
		os.Exit(1)
	}
	/*啟用多核*/
	runtime.GOMAXPROCS(runtime.NumCPU())

	/*定義log路徑*/
	logDir := getLogPath()
	if !checkPathExist(logDir) {
		os.Mkdir(logDir, 0664)
	}
	rotatingHandler := logger.NewRotatingHandler(logDir, logFileName, 5, 10*1024*1024)
	logger.SetHandlers(logger.Console, rotatingHandler)

	defer logger.Close()
	logger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logger.SetLevel(logger.INFO)
	ftpClient = new(ftp.ServerConn)
	ftpClientfordel = new(ftp.ServerConn)
	/*两个ftp句柄，一个用于删除，一个用于读文件*/
	for {
		ftpClient = handleFtpConn()
		ftpClientfordel = handleFtpConn()
		if nil != ftpClient && nil != ftpClientfordel {
			break
		}
		time.Sleep(5 * time.Second)
	}

	logger.Error("Connect OK!\n")

	PoliceMsg = make(chan []byte, 512)
	keepAliveChann = make(chan bool, 10)
	delFileChan = make(chan string, 512)
	getFtpTicker := time.NewTicker(1 * time.Second)
	getFtpTickerChan := getFtpTicker.C
	/*调用接口*/
	go sendMsg()
	/*删除文件*/
	go handleDelFile()
	for {
		<-getFtpTickerChan
		//ftpClient.List()
		if ftpClientfordel == nil || ftpClient == nil {
			logger.Error("ftpClientfordel is nil!")
			/*把 keepAliveChann 放到本协程，执行不到，因为 continue*/
			keepAliveChann <- true
			continue
		}
		logger.Info("+++++++++++++++++")
		gList, err := ftpClient.NameList(workDir)
		if err != nil {
			logger.Error("ftpClient.NameList Error:", err.Error())
			if ftpClient != nil {
				ftpClient.Quit()
			}
			if ftpClientfordel != nil {
				ftpClientfordel.Quit()
			}
			/*把 keepAliveChann 放到本协程，执行不到*/
			keepAliveChann <- true
			continue
		}
		logger.Info("gList len:", len(gList))
		for k, val := range gList {
			logger.Info("gList :", k, "# ", val)
			tmpStr := strings.Split(val, ".")
			timeStr := strings.Split(tmpStr[0], "_")
			//logger.Info(timeStr)
			/*删除文件名格式错误的文件*/
			if len(timeStr) == 3 {
				timeInt, _ := strconv.Atoi(timeStr[2])
				ttime := time.Since(time.Unix(int64(timeInt/1000), 0))
				logger.Info("Since Time:", "# ", ttime.Seconds())
				if ttime.Seconds() > 600 {
					logger.Info("Delete old file 10min:", "# ", val)
					delFileChan <- val
					continue
				}
			} else {
				delFileChan <- val
				continue
			}
			if !strings.HasSuffix(val, "txt") {
				delFileChan <- val
				continue
			}
			handleFile(val)
		}
	}
}
func handleDelFile() {
	for {
		select {
		case delFile := <-delFileChan:
			fullFileName := "/alarms/" + delFile
			//time.Sleep(3 * time.Second)
			err := ftpClientfordel.Delete(fullFileName)
			if err != nil {
				logger.Error("Delete Error:", err.Error(), "# ", delFile)
			}
			logger.Info("Delete File Success:", "# ", delFile)
			//continue
		case <-keepAliveChann:
			logger.Error("reConnect!\n")
			ftpClient = handleFtpConn()
			if nil == ftpClient {
				logger.Error("ftpClient reConnect false!\n")
			} else {
				logger.Error("ftpClient reConnect OK!\n")
			}
			ftpClientfordel = handleFtpConn()
			if nil == ftpClientfordel {
				logger.Error("ftpClientfordel reConnect false!\n")
			} else {
				logger.Error("ftpClientfordel reConnect OK!\n")
			}
		}

	}
}
func handleFile(fileName string) {
	byteData := getFileData(fileName)
	logger.Info("getFileLen:", len(byteData), "# ", fileName)
	if len(byteData) != 0 {
		logger.Info("send ok", "# ", fileName)
		PoliceMsg <- byteData
		delFileChan <- fileName
		/*
			fullFileName := "/alarms/" + fileName
			err := ftpClient.Delete(fullFileName)
			if err != nil {
				logger.Error("Delete Err:", err.Error(), "# ", fullFileName)
				//os.Exit(1)
			} else {
				logger.Error("Delete ok:", "# ", fullFileName)
			}
		*/
	} else {
		logger.Info("send false", "# ", fileName)
	}

}

func getFileData(fileName string) []byte {
	fullFileName := "/alarms/" + fileName
	r, err := ftpClient.Retr(fullFileName)
	if err != nil {
		logger.Error("ftp Read Error1:", fileName, err.Error(), "# ", fullFileName)
		if r != nil {
			r.Close()
		}
		//os.Exit(1)
		return nil
	}
	defer func() {
		if r != nil {
			r.Close()
		}
	}()
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		logger.Error("ftp Read Error2:", fileName, err.Error(), "# ", fullFileName)
		return nil
	}
	return buf
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
		body := bytes.NewReader(rdata)

		go func() {
			client := &http.Client{}
			presp, perr := client.Post(platformAPI, "application/json", body)
			if perr != nil {
				logger.Error(perr.Error())
				return
			}
			defer presp.Body.Close()
			rbody, err := ioutil.ReadAll(presp.Body)
			if err != nil {
				logger.Error(err.Error())
				return
			}
			logger.Info(string(rbody))
			return
		}()
	}
}

func getLogPath() string {
	dir, _ := os.Executable()
	exPath := filepath.Dir(dir)
	sstr := strings.Split(exPath, "/bin")

	logDir := sstr[0] + "/log"
	return logDir
}
func handleFtpConn() *ftp.ServerConn {
	//ftpClient, err = ftp.DialTimeout(ftpAddr, 5*time.Second)

	ftpClient, err := ftp.Dial(ftpAddr, ftp.DialWithTimeout(1*time.Second), ftp.DialWithDisabledEPSV(true))
	if err != nil {
		logger.Error(" ftp.Dial:", err.Error())
		return nil
		//os.Exit(1)
	}
	err = ftpClient.Login(ftpUser, "")
	if err != nil {
		logger.Error(" ftpClient.Login", err.Error())
		return nil
		//os.Exit(1)
	}

	currentDir, err := ftpClient.CurrentDir()
	if err != nil {
		logger.Error("tpClient.CurrentDir", err.Error())
		return nil
		//os.Exit(1)
	}
	//logger.Info(currentDir)
	if currentDir != workDir {
		err = ftpClient.ChangeDir("alarms")
		if err != nil {
			logger.Error("ftpClient.ChangeDir", err.Error())
			return nil
			//os.Exit(1)
		}
	}
	currentDir, err = ftpClient.CurrentDir()
	if err != nil {
		logger.Error(err.Error())
		return nil
		//os.Exit(1)
	}
	logger.Info("currentDir:", currentDir)
	return ftpClient
}
