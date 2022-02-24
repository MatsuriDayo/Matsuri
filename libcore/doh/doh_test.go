package doh

import (
	"fmt"
	"testing"
)

func TestLookupManyDoH(t *testing.T) {
	test := func(domain string) {
		a, b := LookupManyDoH(domain, 1)
		fmt.Println("A", domain, a, b)

		a, b = LookupManyDoH(domain, 28)
		fmt.Println("AAAA", domain, a, b)
	}
	testTxt := func(domain string) {
		a, b := LookupManyDoH(domain, 16)
		fmt.Println("TXT", domain, a, b)
	}

	testTxt("nachonekodayo.sekai.icu")

	test("pixiv.net")
	test("google.com")
	test("youtube.com")
	test("twitter.com")
	test("reddit.com")
	test("instagram.com")
	test("facebook.com")
	test("tiktok.com")
	test("baidu.com")
	test("qq.com")
	test("taobao.com")
	test("bilibili.com")
	test("xn--fiq228c.tw")
	test("中文.tw")
	test("qq.中国")
	test("腾讯.中国")
}
