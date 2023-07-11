package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

func main() {
	postData := `{"foo": "hello world!"}`
	req, err := http.NewRequest(http.MethodPost, "https://qtopie.github.io", bytes.NewBufferString(postData))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")

	q := req.URL.Query()
	q.Add("key1", "abc")
	q.Add("key2", "123")
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(respData))
}
