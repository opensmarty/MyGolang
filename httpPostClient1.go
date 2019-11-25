/*
 *  http client 端
 */
package main

import (
    "fmt"
    "net/http"
    "net/url"
    "strconv"
    "strings"
)

func sendPost1() {
    data := make(url.Values)
    data["name"] = []string{"rnben"}

    res, err := http.PostForm("http://127.0.0.1/tpost", data)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    defer res.Body.Close()
    fmt.Println("post send success")
}

func sendPost2() {
    apiUrl := "http://127.0.0.1"
    resource := "/tpost"
    data := url.Values{}
    data.Set("name", "rnben")

    u, _ := url.ParseRequestURI(apiUrl)
    u.Path = resource
    urlStr := u.String() // "http://127.0.0.1/tpost"

    client := &http.Client{}
    r, _ := http.NewRequest("POST", urlStr, strings.NewReader(data.Encode())) // URL-encoded payload
    r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

    resp, err := client.Do(r)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    defer resp.Body.Close()
    fmt.Println("post send success")

}

func main() {
    sendPost1()
    sendPost2()
}
