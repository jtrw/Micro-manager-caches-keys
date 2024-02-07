package main

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log"
	"math/rand"
	server "micro-manager-redis/app/server"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestMainRun(t *testing.T) {
	// Mock options
	opts := Options{
		Listen:         ":8080",
		PinSize:        5,
		MaxExpire:      24 * time.Hour,
		MaxPinAttempts: 3,
		WebRoot:        "./web",
		Secret:         "123",
		RedisUrl:       "localhost:6379",
		Database:       3,
		RedisPass:      "password",
		AuthLogin:      "admin",
		AuthPassword:   "admin",
	}

	// Mock context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a mock server instance
	srv := server.Server{
		Listen:         opts.Listen,
		PinSize:        opts.PinSize,
		MaxExpire:      opts.MaxExpire,
		MaxPinAttempts: opts.MaxPinAttempts,
		WebRoot:        opts.WebRoot,
		WebFS:          webFS,
		Secret:         opts.Secret,
		Version:        revision,
		AuthLogin:      opts.AuthLogin,
		AuthPassword:   opts.AuthPassword,
	}

	// Run the server (this will block, so we run it in a goroutine)
	go func() {
		err := srv.Run(ctx)
		log.Printf("[ERROR]!!! failed, %+v", err)
		//assert.NoError(t, err)
	}()

	// Simulate an interrupt signal to stop the server
	cancel()
}

func Test_main(t *testing.T) {
	port := 40000 + int(rand.Int31n(10000))
	os.Args = []string{"app", "--secret=123", "--listen=" + "localhost:" + strconv.Itoa(port)}

	done := make(chan struct{})
	go func() {
		<-done
		e := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.NoError(t, e)
	}()

	finished := make(chan struct{})
	go func() {
		main()
		close(finished)
	}()

	// defer cleanup because require check below can fail
	defer func() {
		close(done)
		<-finished
	}()

	waitForHTTPServerStart(port)
	time.Sleep(time.Second)
	client := &http.Client{}

	{
		url := fmt.Sprintf("http://localhost:%d/ping", port)
		req, err := getRequest(url)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, "pong", string(body))
	}
}

func waitForHTTPServerStart(port int) {
	// wait for up to 10 seconds for server to start before returning it
	client := http.Client{Timeout: time.Second}
	for i := 0; i < 100; i++ {
		time.Sleep(time.Millisecond * 100)
		if resp, err := client.Get(fmt.Sprintf("http://localhost:%d/ping", port)); err == nil {
			_ = resp.Body.Close()
			return
		}
	}
}

func getRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	return req, err
}
