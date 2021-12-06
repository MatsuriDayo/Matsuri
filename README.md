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

## Documents

https://sagernet.org

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

## Credits

#### SagerNet

`The original app of Matsuri.`

Licensed under GPLv3

[SagerNet]: https://github.com/SagerNet/SagerNet/blob/master/LICENSE

#### shadowsocks-android

`The first professional proxy application on native android.`

Licensed under [GPLv3 or later][shadowsocks-android]

[shadowsocks-android]: https://github.com/shadowsocks/shadowsocks-android/blob/master/LICENSE

#### v2ray-core

`A unified platform for anti-censorship, as the core, providing routing, DNS, and more for SN.`

Licensed under [MIT][v2ray-core]

[v2ray-core]: https://github.com/shadowsocks/shadowsocks-android/blob/master/LICENSE

#### clash (OPEN SOURCE version)

`Provides built-in shadowsocks plugins and SSR support for SN.`

Licensed under [GPLv3][clash]

[clash]: https://github.com/Dreamacro/clash/blob/master/LICENSE

#### Plugins

<ul>
    <li><a href="https://github.com/p4gefau1t/trojan-go/blob/master/LICENSE">p4gefau1t/Trojan-Go</a>: <code>GPL 3.0</code></li>
    <li><a href="https://github.com/klzgrad/naiveproxy/blob/master/LICENSE">klzgrad/naiveproxy</a>:  <code>BSD-3-Clause License</code></li>
    <li><a href="https://github.com/esrrhs/pingtunnel/blob/master/LICENSE">esrrhs/pingtunnel</a>:  <code>MIT</code></li>
    <li><a href="https://github.com/WireGuard/wireguard-go/blob/master/LICENSE">WireGuard/wireguard-go</a>:  <code>MIT</code></li>

</ul>
