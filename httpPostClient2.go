/*
 *  如果你不想用现成的http包，那么下面这段代码就是直接通过HTTP协议实现post请求
 */
package main

import (
	"fmt"
	"net"
)

func main() {
	//因为post方法属于HTTP协议，HTTP协议以TCP为基础，所以先建立一个
	//TCP连接，通过这个TCP连接来发送我们的post请求
	conn, err := net.Dial("tcp", "127.0.0.1:80")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer conn.Close()

	//构造post请求
	var post string
	post += "POST /postpage HTTP/1.1\r\n"
	post += "Content-Type: application/x-www-form-urlencoded\r\n"
	post += "Content-Length: 37\r\n"
	post += "Connection: keep-alive\r\n"
	post += "Accept-Language: zh-CN,zh;q=0.8,en;q=0.6\r\n"
	post += "\r\n"
	post += "key=this is key&value=this is value\r\n"

	if _, err := conn.Write([]byte(post)); err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("post send success.")
}
