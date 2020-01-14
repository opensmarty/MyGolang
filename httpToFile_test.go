package main

import (
	"bytes"
	"encoding/json"
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
	ss := &CallPolice{}
	ss.AreaCode = "0997"
	ss.Bjhm = GetRandomString(5)
	ss.Fscs = "333"
	ss.Jkxh = "1"
	ss.Lat = "113.9321"
	ss.Lng = "48.0003"
	ss.Qhfw = "777"
	ss.SthyLsh = "201910230001"
	ss.Zxh = "10001"

	jss, err := json.Marshal(ss)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	strjss := url.QueryEscape(string(jss))
	body := bytes.NewReader([]byte(strjss))
	client := &http.Client{}
	presp, perr := client.Post("http://127.0.0.1:5656/Location/callPolice", "application/x-www-form-urlencoded", body)
	if perr != nil {
		fmt.Println(perr.Error())
		return
	}
	defer presp.Body.Close()
	rbody, err := ioutil.ReadAll(presp.Body)
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
