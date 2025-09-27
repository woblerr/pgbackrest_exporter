package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000))
	if err != nil {
		t.Errorf("\nGet error during generate random int value:\n%v", err)
	}
	port := 50000 + int(n.Int64())
	os.Args = []string{"pgbackrest_exporter", "--web.listen-address=:" + strconv.Itoa(port)}
	finished := make(chan struct{})
	go func() {
		main()
		close(finished)
	}()
	time.Sleep(time.Second)
	urlList := []string{
		fmt.Sprintf("http://localhost:%d/metrics", port),
		fmt.Sprintf("http://localhost:%d/", port),
	}
	for _, url := range urlList {
		t.Run(url, func(t *testing.T) {
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("\nGet error during GET:\n%v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatalf("\nGet bad response code for %s:\n%v\nwant:\n%v", url, resp.StatusCode, 200)
			}
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("\nGet error during read resp body for %s:\n%v", url, err)
			}
			if len(b) == 0 {
				t.Fatalf("\nGet zero body for %s:\n%s", url, string(b))
			}
		})
	}
}
