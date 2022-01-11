package libcore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"

	"github.com/Dreamacro/clash/transport/socks5"
	"github.com/v2fly/v2ray-core/v5/common/buf"
)

type HTTPClient interface {
	RestrictedTLS()
	ModernTLS()
	PinnedTLS12()
	PinnedSHA256(sumHex string)
	TrySocks5(port int32)
	KeepAlive()
	NewRequest() HTTPRequest
	Close()
}

type HTTPRequest interface {
	SetURL(link string) error
	SetMethod(method string)
	SetHeader(key string, value string)
	SetContent(content []byte)
	SetContentString(content string)
	RandomUserAgent()
	SetUserAgent(userAgent string)
	Execute() (HTTPResponse, error)
}

type HTTPResponse interface {
	GetContent() ([]byte, error)
	GetContentString() (string, error)
	WriteTo(path string) error
}

var (
	_ HTTPClient   = (*httpClient)(nil)
	_ HTTPRequest  = (*httpRequest)(nil)
	_ HTTPResponse = (*httpResponse)(nil)
)

type httpClient struct {
	tls       tls.Config
	client    http.Client
	transport http.Transport
}

func NewHttpClient() HTTPClient {
	client := new(httpClient)
	client.client.Transport = &client.transport
	client.transport.TLSClientConfig = &client.tls
	client.transport.DisableKeepAlives = true
	return client
}

func (c *httpClient) ModernTLS() {
	c.tls.MinVersion = tls.VersionTLS12
	c.tls.CipherSuites = map1(tls.CipherSuites(), func(it *tls.CipherSuite) uint16 { return it.ID })
}

func (c *httpClient) RestrictedTLS() {
	c.tls.MinVersion = tls.VersionTLS13
	c.tls.CipherSuites = map1(filter1(tls.CipherSuites(), func(it *tls.CipherSuite) bool {
		return contians1(it.SupportedVersions, uint16(tls.VersionTLS13))
	}), func(it *tls.CipherSuite) uint16 {
		return it.ID
	})
}

func (c *httpClient) PinnedTLS12() {
	c.tls.MinVersion = tls.VersionTLS12
	c.tls.MaxVersion = tls.VersionTLS12
}

func (c *httpClient) PinnedSHA256(sumHex string) {
	c.tls.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		for _, rawCert := range rawCerts {
			certSum := sha256.Sum256(rawCert)
			if sumHex == hex.EncodeToString(certSum[:]) {
				return nil
			}
		}
		return newError("pinned sha256 sum mismatch")
	}
}

func (c *httpClient) TrySocks5(port int32) {
	dialer := new(net.Dialer)
	c.transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		for {
			socksConn, err := dialer.DialContext(ctx, "tcp", "127.0.0.1:"+strconv.Itoa(int(port)))
			if err != nil {
				break
			}
			_, err = socks5.ClientHandshake(socksConn, socks5.ParseAddr(addr), socks5.CmdConnect, nil)
			if err != nil {
				break
			}
			return socksConn, err
		}
		return dialer.DialContext(ctx, network, addr)
	}
}

func (c *httpClient) KeepAlive() {
	c.transport.ForceAttemptHTTP2 = true
	c.transport.DisableKeepAlives = false
}

func (c *httpClient) NewRequest() HTTPRequest {
	req := &httpRequest{httpClient: c}
	req.request = http.Request{
		Method: "GET",
		Header: http.Header{},
	}
	return req
}

func (c *httpClient) Close() {
	c.transport.CloseIdleConnections()
}

type httpRequest struct {
	*httpClient
	request http.Request
}

func (r *httpRequest) SetURL(link string) (err error) {
	r.request.URL, err = url.Parse(link)
	return
}

func (r *httpRequest) SetMethod(method string) {
	r.request.Method = method
}

func (r *httpRequest) SetHeader(key string, value string) {
	r.request.Header.Set(key, value)
}

func (r *httpRequest) RandomUserAgent() {
	r.request.Header.Set("User-Agent", fmt.Sprintf("curl/7.%d.%d", rand.Int()%54, rand.Int()%2))
}

func (r *httpRequest) SetUserAgent(userAgent string) {
	r.request.Header.Set("User-Agent", userAgent)
}

func (r *httpRequest) SetContent(content []byte) {
	buffer := bytes.Buffer{}
	buffer.Write(content)
	r.request.Body = io.NopCloser(bytes.NewReader(buffer.Bytes()))
	r.request.ContentLength = int64(len(content))
}

func (r *httpRequest) SetContentString(content string) {
	r.SetContent([]byte(content))
}

func (r *httpRequest) Execute() (HTTPResponse, error) {
	response, err := r.client.Do(&r.request)
	if err != nil {
		return nil, err
	}
	httpResp := &httpResponse{Response: response}
	if response.StatusCode != http.StatusOK {
		return nil, errors.New(httpResp.errorString())
	}
	return httpResp, nil
}

type httpResponse struct {
	*http.Response

	getContentOnce sync.Once
	content        []byte
	contentError   error
}

func (h *httpResponse) errorString() string {
	content, err := h.GetContentString()
	if err != nil {
		return fmt.Sprint("HTTP ", h.Status)
	}
	return fmt.Sprint("HTTP ", h.Status, ": ", content)
}

func (h *httpResponse) GetContent() ([]byte, error) {
	h.getContentOnce.Do(func() {
		defer h.Body.Close()
		h.content, h.contentError = ioutil.ReadAll(h.Body)
	})
	return h.content, h.contentError
}

func (h *httpResponse) GetContentString() (string, error) {
	content, err := h.GetContent()
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (h *httpResponse) WriteTo(path string) error {
	defer h.Body.Close()
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	buffer := buf.StackNew()
	defer buffer.Release()
	_, err = io.CopyBuffer(file, h.Body, buffer.Extend(buf.Size))
	return err
}

// TODO remove when go 1.18

func map1(arr []*tls.CipherSuite, block func(it *tls.CipherSuite) uint16) []uint16 {
	var retArr []uint16
	for index := range arr {
		retArr = append(retArr, block(arr[index]))
	}
	return retArr
}

func filter1(arr []*tls.CipherSuite, block func(it *tls.CipherSuite) bool) []*tls.CipherSuite {
	var retArr []*tls.CipherSuite
	for _, it := range arr {
		if block(it) {
			retArr = append(retArr, it)
		}
	}
	return retArr
}

func contians1(arr []uint16, target uint16) bool {
	for i := range arr {
		if target == arr[i] {
			return true
		}
	}
	return false
}
