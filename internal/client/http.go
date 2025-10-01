package client

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

type BrowserImpersonation string

const (
	BrowserNone          BrowserImpersonation = "none"
	BrowserChrome        BrowserImpersonation = "chrome"
	BrowserChromeAndroid BrowserImpersonation = "chrome_android"
	BrowserSafari        BrowserImpersonation = "safari"
	BrowserSafariIOS     BrowserImpersonation = "safari_ios"
	BrowserEdge          BrowserImpersonation = "edge"
	BrowserFirefox       BrowserImpersonation = "firefox"
)

var UserAgents = map[string]string{
	"chrome":         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"chrome_android": "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
	"firefox":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"safari":         "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
	"safari_ios":     "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1",
	"edge":           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
}

type ProxyRotator struct {
	proxies []string
	current int
	mu      sync.Mutex
}

func NewProxyRotator(proxies []string) *ProxyRotator {
	return &ProxyRotator{
		proxies: proxies,
		current: 0,
	}
}

func LoadProxiesFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open proxies file: %w", err)
	}
	defer file.Close()

	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		proxies = append(proxies, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read proxies file: %w", err)
	}

	if len(proxies) == 0 {
		return nil, fmt.Errorf("no valid proxies found in file")
	}

	return proxies, nil
}

func (pr *ProxyRotator) Next() string {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if len(pr.proxies) == 0 {
		return ""
	}

	proxy := pr.proxies[pr.current]
	pr.current = (pr.current + 1) % len(pr.proxies)
	return proxy
}

func (pr *ProxyRotator) Random() string {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if len(pr.proxies) == 0 {
		return ""
	}

	return pr.proxies[rand.Intn(len(pr.proxies))]
}

type HTTPClient struct {
	client        *http.Client
	userAgent     string
	proxyRotator  *ProxyRotator
	singleProxy   string
	timeout       time.Duration
	verifySSL     bool
	allowRedirect bool
}

type ClientConfig struct {
	Timeout       int
	VerifySSL     bool
	AllowRedirect bool
	Impersonate   BrowserImpersonation
	Proxy         string
	ProxyFile     string
}

func NewHTTPClient(config ClientConfig) (*HTTPClient, error) {
	timeout := time.Duration(config.Timeout) * time.Second

	userAgent := UserAgents[string(config.Impersonate)]
	if userAgent == "" {
		userAgent = UserAgents["chrome"]
	}

	client := &HTTPClient{
		userAgent:     userAgent,
		timeout:       timeout,
		verifySSL:     config.VerifySSL,
		allowRedirect: config.AllowRedirect,
	}

	var transport *http.Transport
	if config.ProxyFile != "" {
		proxies, err := LoadProxiesFromFile(config.ProxyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load proxies: %w", err)
		}
		client.proxyRotator = NewProxyRotator(proxies)

		proxyURL := client.proxyRotator.Next()
		transport, err = createTransport(proxyURL, config.VerifySSL)
		if err != nil {
			return nil, err
		}
	} else if config.Proxy != "" {
		client.singleProxy = config.Proxy
		var err error
		transport, err = createTransport(config.Proxy, config.VerifySSL)
		if err != nil {
			return nil, err
		}
	} else {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !config.VerifySSL,
			},
		}
	}

	var checkRedirect func(req *http.Request, via []*http.Request) error
	if !config.AllowRedirect {
		checkRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	client.client = &http.Client{
		Transport:     transport,
		Timeout:       timeout,
		CheckRedirect: checkRedirect,
	}

	return client, nil
}

func createTransport(proxyStr string, verifySSL bool) (*http.Transport, error) {
	if proxyStr == "" {
		return &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !verifySSL,
			},
		}, nil
	}

	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	if proxyURL.Scheme == "socks5" {
		dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, nil, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
		}

		return &http.Transport{
			Dial: dialer.Dial,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !verifySSL,
			},
		}, nil
	}

	return &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !verifySSL,
		},
	}, nil
}

func (c *HTTPClient) RotateProxy() error {
	if c.proxyRotator == nil {
		return nil
	}

	proxyURL := c.proxyRotator.Next()
	transport, err := createTransport(proxyURL, c.verifySSL)
	if err != nil {
		return err
	}

	c.client.Transport = transport
	return nil
}

func (c *HTTPClient) Get(url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.userAgent)

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return c.client.Do(req)
}

func (c *HTTPClient) Post(url string, headers map[string]string, body string) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.userAgent)

	if headers["Content-Type"] == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return c.client.Do(req)
}

func ReadResponseBody(resp *http.Response) (string, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
