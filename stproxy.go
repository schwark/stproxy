package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/schwark/go-ssdp"
)

type Configuration struct {
	Hosts map[string]string
	Port  string
}

type baseHandle struct{}

var (
	hostProxy map[string]*httputil.ReverseProxy = map[string]*httputil.ReverseProxy{}
	config    Configuration
)

func read_config(config string) Configuration {
	configuration := Configuration{}
	configuration.Port = "8081" // default port
	configuration.Hosts = map[string]string{
		"1": "https://www.alarm.com",
	} // default mapping
	file, err := os.Open(config)
	if err != nil {
		log.Println("warning: no config file, using defaults : ", err)
	} else {
		defer file.Close()
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&configuration)
		if err != nil {
			log.Fatal("error:", err)
		}
		log.Printf("Read proxy configuration: %v\n", configuration)
	}
	return configuration
}

func (h *baseHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(r.URL.String(), "/", 3)
	prefix := parts[1]
	rest := parts[2]
	r.URL, _ = url.Parse(rest)

	if target, ok := config.Hosts[prefix]; ok {
		log.Println(r.Method + " " + config.Hosts[prefix] + " " + rest)

		if fn, ok := hostProxy[prefix]; ok {
			fn.ServeHTTP(w, r)
			return
		}

		remoteUrl, err := url.Parse(target)
		if err != nil {
			log.Println("target parse fail:", err)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(remoteUrl)
		hostProxy[prefix] = proxy
		proxy.ServeHTTP(w, r)
		return
	}
	w.Write([]byte("403: Host forbidden for path prefix " + prefix))
}

func ssdp_server(wg *sync.WaitGroup) {
	defer wg.Done()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	st := "urn:SmartThingsCommunity:device:GenericProxy:1"
	usn := "uuid:de8a5619-2603-40d1-9e21-1967952d7f86"
	loc := "http://127.0.0.1:" + config.Port + "/"
	srv := ""
	maxAge := 1800
	ai := 100

	ssdp.Logger = log.New(os.Stderr, "[SSDP] ", log.LstdFlags)

	ad, err := ssdp.Advertise(st, usn, loc, srv, maxAge)
	if err != nil {
		log.Fatal(err)
	}
	var aliveTick <-chan time.Time
	if ai > 0 {
		aliveTick = time.Tick(time.Duration(ai) * time.Second)
	} else {
		aliveTick = make(chan time.Time)
	}

loop:
	for {
		select {
		case <-aliveTick:
			ad.Alive()
		case <-quit:
			break loop
		}
	}
	ad.Bye()
	ad.Close()
}

func proxy_server(wg *sync.WaitGroup) {
	defer wg.Done()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	h := &baseHandle{}
	http.Handle("/", h)

	server := &http.Server{
		Addr:    ":8081",
		Handler: h,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Print("Proxy Server Started")

	<-quit
	log.Print("Proxy Server Stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		// extra handling here
		cancel()
	}()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Proxy Server Shutdown Failed:%+v", err)
	}
	log.Print("Proxy Server Exited Properly")
}

func main() {
	var wg sync.WaitGroup

	c := flag.String("d", "config.json", "full path to config.json")
	h := flag.Bool("h", false, "show help")
	flag.Parse()
	if *h {
		flag.Usage()
		return
	}
	config = read_config(*c)

	wg.Add(2)
	go ssdp_server(&wg)
	go proxy_server(&wg)

	log.Println("Server initialized...")
	wg.Wait()
	log.Println("Server Shutdown!")
}
