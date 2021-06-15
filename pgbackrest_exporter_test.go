package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	port := 50000 + int(rand.Int31n(100))
	os.Args = []string{"pgbackrest_exporter", "--prom.port=" + strconv.Itoa(port)}
	finished := make(chan struct{})
	go func() {
		main()
		close(finished)
	}()
	time.Sleep(time.Second)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	if err != nil {
		t.Errorf("\nGet error during GET:\n%v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", resp.StatusCode, 200)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("\nGet error during read resp body:\n%v", err)
	}
	if len(string(b)) == 0 {
		t.Errorf("\nGet zero body:\n%s", string(b))
	}
}
