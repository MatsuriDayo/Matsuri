# Matsuri (茉莉) for Android

<img align="right" style="width: 20%" src="https://avatars.githubusercontent.com/u/95122236"/>

[Releases](https://github.com/MatsuriDayo/Matsuri/releases)

[Language: Kotlin](https://github.com/MatsuriDayo/Matsuri/search?l=kotlin)

[License: GPL-3.0](https://www.gnu.org/licenses/gpl-3.0)

A proxy toolchain for Android, written in Kotlin.

# 中文

## 与 SagerNet 的区别

“错误修复和其他改进”

相较于 SagerNet 本软件的功能更少。

其他： https://nekoquq.github.io/posts/0009.html

## Documents & Changelog

https://t.me/Matsuridayo

### Protocols

The application is designed to support some of the proxy protocols bypassing the firewall.

#### Proxy

* SOCKS (4/4a/5)
* HTTP(S)
* SSH
* Shadowsocks
* ShadowsocksR
* VMess
* Trojan
* Trojan-Go ( trojan-go-plugin )
* NaïveProxy ( naive-plugin )
* Hysteria ( hysteria-plugin )
* WireGuard ( wireguard-plugin )

##### ROOT required

* Ping Tunnel ( pingtunnel-plugin )

#### Subscription

* Raw: All widely used formats (base64, clash or origin configuration)
* [Open Online Config](https://github.com/Shadowsocks-NET/OpenOnlineConfig)
* [Shadowsocks SIP008](https://shadowsocks.org/en/wiki/SIP008-Online-Configuration-Delivery.html)

#### Features

* Full basic features
* V2Ray WebSocket browser forwarding
* Option to change the notification update interval
* A Chinese apps scanner (based on dex classpath scanning, so it may be slower)
* Proxy chain
* Advanced routing with outbound profile selection
* Reverse proxy
* Custom config (V2Ray / Trojan-Go)
* Traffic statistics support, including real-time display and cumulative statistics
