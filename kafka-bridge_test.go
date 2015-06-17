package main

import (
    "fmt"
    "io/ioutil"
    "testing"
    "strings"
    "net/http"
    "time"
)

func TestParse(t *testing.T) {

    buf, err := ioutil.ReadFile("test_msg.txt")
    if (err != nil) {
        fmt.Print("Couldn't read content from file.")
    }
    msg := string(buf)

    fmt.Println(msg);
    client := &http.Client{
        Timeout: 5 * time.Second,
    }
    fmt.Println("Creating request.")
    req, err := http.NewRequest("POST", "http://localhost:8080/notify", strings.NewReader("{\"uuid\" : \"test\"}"));

    if (err != nil) {
        fmt.Printf("Error: %v\n", err.Error())
        panic(err)
    }

    fmt.Println("Adding headers.")
//    req.Header.Set("Host", "cms-notifier")
    req.Header.Set("X-Origin-System-Id", "methode-web-pub")

    resp, err := client.Do(req)
    if (err != nil) {
        fmt.Printf("Error: %v\n", err.Error())
        panic(err)
    }

    fmt.Printf("\nResponse: %v\n", resp)
}
