package doh

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

var dohs = []string{
	"https://1.0.0.1/dns-query",
	"https://101.101.101.101/dns-query",
	"https://8.8.4.4/resolve",
	"https://[2001:4860:4860::8844]/dns-query",
	"https://[2606:4700:4700::1111]/dns-query",
	"https://[2620:fe::9]/dns-query",
	"https://149.112.112.112:5053/dns-query",
	"https://9.9.9.9:5053/dns-query",
}

// Return net.IP for A&AAAA
// Return string for other
func LookupManyDoH(domain string, queryType int) (interface{}, error) {
	var good, bad int32

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	in := make(chan string, len(dohs))
	out := make(chan interface{}, 0)

	defer func() {
		cancel()
		close(in)
		close(out)
	}()

	for i := 0; i < len(dohs); i++ {
		go manyWorker(ctx, domain, queryType, in, out, cancel, &good, &bad)
	}

	go func() {
		for _, doh := range dohs {
			in <- doh
		}
	}()

	var err error
	ips := <-out

	if ips == nil {
		err = errors.New("LookupManyDoH: all tries failed")
	} else if a, ok := ips.([]net.IP); ok && len(a) == 0 {
		err = errors.New("LookupManyDoH: empty")
	}
	logrus.Debugln("LookupManyDoH:", domain, queryType, ips, err)

	return ips, err
}

func manyWorker(ctx context.Context, domain string, queryType int, in chan string, out chan interface{}, cancel context.CancelFunc, good, bad *int32) {
	for {
		select {
		case <-ctx.Done():
			return
		case doh, ok := <-in:
			if !ok { // closed
				return
			}

			ret := lookupDoH(ctx, doh, domain, queryType)
			if ret == nil {
				//failed
				if atomic.AddInt32(bad, 1) == int32(len(dohs)) {
					//all failed
					cancel()
					out <- nil
					return
				}
				continue
			}
			if atomic.AddInt32(good, 1) == 1 {
				//first success
				cancel()
				out <- ret
			}
		}
	}
}

var client = &http.Client{
	Transport: &http.Transport{
		IdleConnTimeout:       5 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	},
}

func SetDialContext(f func(ctx context.Context, network, addr string) (net.Conn, error)) {
	client.Transport.(*http.Transport).DialContext = f
}

func lookupDoH(ctx context.Context, doh, domain string, queryType int) interface{} {
	dohURL := doh + "?name=" + domain + "&type=" + strconv.Itoa(queryType)

	req, _ := http.NewRequestWithContext(ctx, "GET", dohURL, nil)
	req.Header.Set("accept", "application/dns-json")
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	v := &struct {
		Answer []struct {
			Type int    `json:"type"`
			Data string `json:"data"`
		}
	}{}

	err = json.NewDecoder(resp.Body).Decode(v)
	if err != nil {
		return nil
	}

	ips := make([]net.IP, 0)

	for _, a := range v.Answer {
		if a.Type == queryType {
			if queryType == 1 || queryType == 28 {
				ips = append(ips, net.ParseIP(a.Data))
			} else {
				return a.Data
			}
		}
	}

	return ips
}
