package main

import (
	//"bytes"
	//"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"time"
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

func main() {
	caller := GetRandomString(5)
   /* 表单的形式提交数据 */
	data := make(url.Values)
	data["area_code"] = []string{"0997"}
	data["jkxh"] = []string{"1"}
	data["bjhm"] = []string{caller}
	data["sthy_lsh"] = []string{"201910230001"}
	data["zxh"] = []string{"10001"}
	data["fscs"] = []string{"333"}
	data["qhfw"] = []string{"777"}
	data["lng"] = []string{"48.0003"}
	data["lat"] = []string{"113.9321"}
	client := &http.Client{}
 
	presp, perr := client.PostForm("http://36.189.xxx.xxx:48000/SG_sns/index.php?app=api&mod=SharXj&act=parseShortNumber", data)
	if perr != nil {
		fmt.Println(perr.Error())
		return
	}
	defer presp.Body.Close()
	rbody, err := ioutil.ReadAll(presp.Body)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(string(rbody))
}
func GetRandomString(len_random int) string {
	//str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	str := "0123456789"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < len_random; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}
