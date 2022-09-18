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

	"github.com/koron/go-ssdp"
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

func read_config() Configuration {
	configuration := Configuration{}
	configuration.Port = "8081" // default port
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatal("error:", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configuration)
	if err != nil {
		log.Fatal("error:", err)
	}
	log.Printf("Read proxy configuration: %v\n", configuration)
	return configuration
}

func (h *baseHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(r.URL.String(), "/", 3)
	prefix := parts[1]
	rest := parts[2]
	r.URL, _ = url.Parse(rest)

	if fn, ok := hostProxy[prefix]; ok {
		fn.ServeHTTP(w, r)
		return
	}

	if target, ok := config.Hosts[prefix]; ok {
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

func ssdp_server(wg *sync.WaitGroup, quit chan os.Signal) {
	defer wg.Done()

	st := flag.String("st", "urn:SmartThingsCommunity:device:GenericProxy:1", "ST: Type")
	usn := flag.String("usn", "uuid:de8a5619-2603-40d1-9e21-1967952d7f86", "USN: ID")
	loc := flag.String("loc", "http://127.0.0.1:8081/", "LOCATION: location header")
	srv := flag.String("srv", "", "SERVER:  server header")
	maxAge := flag.Int("maxage", 1800, "cache control, max-age")
	ai := flag.Int("ai", 10, "alive interval")
	v := flag.Bool("v", true, "verbose mode")
	h := flag.Bool("h", false, "show help")
	flag.Parse()
	if *h {
		flag.Usage()
		return
	}
	if *v {
		ssdp.Logger = log.New(os.Stderr, "[SSDP] ", log.LstdFlags)
	}

	ad, err := ssdp.Advertise(*st, *usn, *loc, *srv, *maxAge)
	if err != nil {
		log.Fatal(err)
	}
	var aliveTick <-chan time.Time
	if *ai > 0 {
		aliveTick = time.Tick(time.Duration(*ai) * time.Second)
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

func proxy_server(wg *sync.WaitGroup, quit chan os.Signal) {
	defer wg.Done()

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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	config = read_config()

	wg.Add(2)
	go ssdp_server(&wg, quit)
	go proxy_server(&wg, quit)

	log.Println("Server initialized...")
	wg.Wait()
	log.Println("Server Shutdown!")
}
