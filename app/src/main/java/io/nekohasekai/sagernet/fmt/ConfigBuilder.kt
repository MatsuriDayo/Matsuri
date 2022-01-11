/******************************************************************************
 *                                                                            *
 * Copyright (C) 2021 by nekohasekai <contact-sagernet@sekai.icu>             *
 *                                                                            *
 * This program is free software: you can redistribute it and/or modify       *
 * it under the terms of the GNU General Public License as published by       *
 * the Free Software Foundation, either version 3 of the License, or          *
 *  (at your option) any later version.                                       *
 *                                                                            *
 * This program is distributed in the hope that it will be useful,            *
 * but WITHOUT ANY WARRANTY; without even the implied warranty of             *
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the              *
 * GNU General Public License for more details.                               *
 *                                                                            *
 * You should have received a copy of the GNU General Public License          *
 * along with this program. If not, see <http://www.gnu.org/licenses/>.       *
 *                                                                            *
 ******************************************************************************/

package io.nekohasekai.sagernet.fmt

import android.os.Build
import cn.hutool.core.util.NumberUtil
import com.github.shadowsocks.plugin.PluginConfiguration
import com.github.shadowsocks.plugin.PluginManager
import io.nekohasekai.sagernet.IPv6Mode
import io.nekohasekai.sagernet.Key
import io.nekohasekai.sagernet.bg.VpnService
import io.nekohasekai.sagernet.database.DataStore
import io.nekohasekai.sagernet.database.ProxyEntity
import io.nekohasekai.sagernet.database.SagerDatabase
import io.nekohasekai.sagernet.fmt.V2rayBuildResult.IndexEntity
import io.nekohasekai.sagernet.fmt.gson.gson
import io.nekohasekai.sagernet.fmt.http.HttpBean
import io.nekohasekai.sagernet.fmt.internal.ChainBean
import io.nekohasekai.sagernet.fmt.shadowsocks.ShadowsocksBean
import io.nekohasekai.sagernet.fmt.shadowsocksr.ShadowsocksRBean
import io.nekohasekai.sagernet.fmt.socks.SOCKSBean
import io.nekohasekai.sagernet.fmt.ssh.SSHBean
import io.nekohasekai.sagernet.fmt.trojan.TrojanBean
import io.nekohasekai.sagernet.fmt.v2ray.StandardV2RayBean
import io.nekohasekai.sagernet.fmt.v2ray.V2RayConfig
import io.nekohasekai.sagernet.fmt.v2ray.V2RayConfig.*
import io.nekohasekai.sagernet.fmt.v2ray.VMessBean
import io.nekohasekai.sagernet.fmt.wireguard.WireGuardBean
import io.nekohasekai.sagernet.ktx.isIpAddress
import io.nekohasekai.sagernet.ktx.mkPort
import io.nekohasekai.sagernet.utils.PackageCache
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

const val TAG_SOCKS = "socks"
const val TAG_HTTP = "http"
const val TAG_TRANS = "trans"

const val TAG_AGENT = "proxy"
const val TAG_DIRECT = "direct"
const val TAG_BYPASS = "bypass"
const val TAG_BLOCK = "block"

const val TAG_DNS_IN = "dns-in"
const val TAG_DNS_OUT = "dns-out"

const val TAG_API_IN = "api-in"

const val LOCALHOST = "127.0.0.1"
const val IP6_LOCALHOST = "::1"

class V2rayBuildResult(
    var config: String,
    var index: List<IndexEntity>,
    var requireWs: Boolean,
    var wsPort: Int,
    var outboundTags: List<String>,
    var outboundTagsCurrent: List<String>,
    var outboundTagsAll: Map<String, ProxyEntity>,
    var bypassTag: String,
    var observatoryTags: Set<String>,
    val dumpUid: Boolean,
    val alerts: List<Pair<Int, String>>,
) {
    data class IndexEntity(var chain: LinkedHashMap<Int, ProxyEntity>)
}

fun buildV2RayConfig(
    proxy: ProxyEntity, forTest: Boolean = false
): V2rayBuildResult {

    val outboundTags = ArrayList<String>()
    val outboundTagsCurrent = ArrayList<String>()
    val outboundTagsAll = HashMap<String, ProxyEntity>()
    val globalOutbounds = ArrayList<String>()

    fun ProxyEntity.resolveChain(): MutableList<ProxyEntity> {
        val bean = requireBean()
        if (bean is ChainBean) {
            val beans = SagerDatabase.proxyDao.getEntities(bean.proxies)
            val beansMap = beans.associateBy { it.id }
            val beanList = ArrayList<ProxyEntity>()
            for (proxyId in bean.proxies) {
                val item = beansMap[proxyId] ?: continue
                beanList.addAll(item.resolveChain())
            }
            return beanList.asReversed()
        }
        return mutableListOf(this)
    }

    val proxies = proxy.resolveChain()
    val extraRules = if (forTest) listOf() else SagerDatabase.rulesDao.enabledRules()
    val extraProxies = if (forTest) mapOf() else SagerDatabase.proxyDao.getEntities(extraRules.mapNotNull { rule ->
        rule.outbound.takeIf { it > 0 && it != proxy.id }
    }.toHashSet().toList()).map { it.id to it.resolveChain() }.toMap()

    val allowAccess = DataStore.allowAccess
    val bind = if (!forTest && allowAccess) "0.0.0.0" else LOCALHOST

    val remoteDns = DataStore.remoteDns.split("\n")
        .mapNotNull { dns -> dns.trim().takeIf { it.isNotBlank() && !it.startsWith("#") } }
    var directDNS = DataStore.directDns.split("\n")
        .mapNotNull { dns -> dns.trim().takeIf { it.isNotBlank() && !it.startsWith("#") } }
    val enableDnsRouting = DataStore.enableDnsRouting
    val useFakeDns = DataStore.enableFakeDns
    val trafficSniffing = DataStore.trafficSniffing
    val indexMap = ArrayList<IndexEntity>()
    var requireWs = false
    val requireHttp = !forTest && DataStore.requireHttp
    val requireTransproxy = if (forTest) false else DataStore.requireTransproxy
    val ipv6Mode = if (forTest) IPv6Mode.ENABLE else DataStore.ipv6Mode
    val resolveDestination = DataStore.resolveDestination
    val destinationOverride = DataStore.destinationOverride
    val trafficStatistics = !forTest && DataStore.profileTrafficStatistics

    var currentDomainStrategy = when {
        !resolveDestination -> "AsIs"
        ipv6Mode == IPv6Mode.DISABLE -> "UseIPv4"
        ipv6Mode == IPv6Mode.PREFER -> "PreferIPv6"
        ipv6Mode == IPv6Mode.ONLY -> "UseIPv6"
        else -> "PreferIPv4"
    }

    var dumpUid = false
    val alerts = mutableListOf<Pair<Int, String>>()

    return V2RayConfig().apply {

        dns = DnsObject().apply {
            hosts = DataStore.hosts.split("\n")
                .filter { it.isNotBlank() }
                .associate { it.substringBefore(" ") to it.substringAfter(" ") }
                .toMutableMap()
            servers = mutableListOf()

            servers.addAll(remoteDns.map {
                DnsObject.StringOrServerObject().apply {
                    valueY = DnsObject.ServerObject().apply {
                        address = it
                    }
                }
            })

            disableFallbackIfMatch = true

            if (useFakeDns) {
                fakedns = mutableListOf()
                fakedns.add(FakeDnsObject().apply {
                    ipPool = "${VpnService.FAKEDNS_VLAN4_CLIENT}/15"
                    poolSize = 65535
                })
                if (ipv6Mode != IPv6Mode.DISABLE) {
                    fakedns.add(FakeDnsObject().apply {
                        ipPool = "${VpnService.FAKEDNS_VLAN6_CLIENT}/18"
                        poolSize = 65535
                    })
                }
            }

            when (ipv6Mode) {
                IPv6Mode.DISABLE -> {
                    queryStrategy = "UseIPv4"
                }
                IPv6Mode.ONLY -> {
                    queryStrategy = "UseIPv6"
                }
            }
        }

        log = LogObject().apply {
            loglevel = if (DataStore.enableLog) "debug" else "error"
        }

        policy = PolicyObject().apply {
            levels = mapOf(
                // dns
                "1" to PolicyObject.LevelPolicyObject().apply {
                    connIdle = 30
                })

            if (trafficStatistics) {
                system = PolicyObject.SystemPolicyObject().apply {
                    statsOutboundDownlink = true
                    statsOutboundUplink = true
                }
            }
        }
        inbounds = mutableListOf()

        if (!forTest) inbounds.add(InboundObject().apply {
            tag = TAG_SOCKS
            listen = bind
            port = DataStore.socksPort
            protocol = "socks"
            settings = LazyInboundConfigurationObject(
                this,
                SocksInboundConfigurationObject().apply {
                    auth = "noauth"
                    udp = true
                })
            if (trafficSniffing || useFakeDns) {
                sniffing = InboundObject.SniffingObject().apply {
                    enabled = true
                    destOverride = when {
                        useFakeDns && !trafficSniffing -> listOf("fakedns")
                        useFakeDns -> listOf("fakedns", "http", "tls")
                        else -> listOf("http", "tls")
                    }
                    metadataOnly = useFakeDns && !trafficSniffing
                    routeOnly = !destinationOverride
                }
            }
        })

        if (requireHttp) {
            inbounds.add(InboundObject().apply {
                tag = TAG_HTTP
                listen = bind
                port = DataStore.httpPort
                protocol = "http"
                settings = LazyInboundConfigurationObject(
                    this,
                    HTTPInboundConfigurationObject().apply {
                        allowTransparent = true
                    })
                if (trafficSniffing || useFakeDns) {
                    sniffing = InboundObject.SniffingObject().apply {
                        enabled = true
                        destOverride = when {
                            useFakeDns && !trafficSniffing -> listOf("fakedns")
                            useFakeDns -> listOf("fakedns", "http", "tls")
                            else -> listOf("http", "tls")
                        }
                        metadataOnly = useFakeDns && !trafficSniffing
                        routeOnly = !destinationOverride
                    }
                }
            })
        }

        if (requireTransproxy) {
            inbounds.add(InboundObject().apply {
                tag = TAG_TRANS
                listen = bind
                port = DataStore.transproxyPort
                protocol = "dokodemo-door"
                settings = LazyInboundConfigurationObject(
                    this,
                    DokodemoDoorInboundConfigurationObject().apply {
                        network = "tcp,udp"
                        followRedirect = true
                    })
                if (trafficSniffing || useFakeDns) {
                    sniffing = InboundObject.SniffingObject().apply {
                        enabled = true
                        destOverride = when {
                            useFakeDns && !trafficSniffing -> listOf("fakedns")
                            useFakeDns -> listOf("fakedns", "http", "tls")
                            else -> listOf("http", "tls")
                        }
                        metadataOnly = useFakeDns && !trafficSniffing
                        routeOnly = !destinationOverride
                    }
                }
                when (DataStore.transproxyMode) {
                    1 -> streamSettings = StreamSettingsObject().apply {
                        sockopt = StreamSettingsObject.SockoptObject().apply {
                            tproxy = "tproxy"
                        }
                    }
                }
            })
        }

        outbounds = mutableListOf()

        // init routing object
        // set rules for wsUseBrowserForwarder and bypass LAN
        routing = RoutingObject().apply {
            domainStrategy = DataStore.domainStrategy

            rules = mutableListOf()

            val wsRules = HashMap<String, RoutingObject.RuleObject>()

            for (proxyEntity in proxies) {
                val bean = proxyEntity.requireBean()

                if (bean is StandardV2RayBean && bean.type == "ws" && bean.wsUseBrowserForwarder == true) {
                    val route = RoutingObject.RuleObject().apply {
                        type = "field"
                        outboundTag = TAG_DIRECT
                        when {
                            bean.host.isIpAddress() -> {
                                ip = listOf(bean.host)
                            }
                            bean.host.isNotBlank() -> {
                                domain = listOf(bean.host)
                            }
                            bean.serverAddress.isIpAddress() -> {
                                ip = listOf(bean.serverAddress)
                            }
                            else -> domain = listOf(bean.serverAddress)
                        }
                    }
                    wsRules[bean.host.takeIf { !it.isNullOrBlank() } ?: bean.serverAddress] = route
                }
            }

            rules.addAll(wsRules.values)

            if (DataStore.bypassLan && (requireHttp || DataStore.bypassLanInCoreOnly)) {
                rules.add(RoutingObject.RuleObject().apply {
                    type = "field"
                    outboundTag = TAG_BYPASS
                    ip = listOf("geoip:private")
                })
            }
        }

        // returns outbound tag
        fun buildChain(
            chainTag: String, profileList: List<ProxyEntity>
        ): String {
            var pastExternal = false // If the past profile is external
            lateinit var pastOutbound: OutboundObject
            lateinit var currentOutbound: OutboundObject
            lateinit var pastInboundTag: String
            val chainMap = LinkedHashMap<Int, ProxyEntity>()
            indexMap.add(IndexEntity(chainMap))
            val chainOutbounds = ArrayList<OutboundObject>()

            // chainOutboundTag: v2ray outbound tag for this chain
            var chainOutboundTag = chainTag

            profileList.forEachIndexed { index, proxyEntity ->
                val bean = proxyEntity.requireBean()
                currentOutbound = OutboundObject()

                // tagOut: v2ray outbound tag for a profile
                // needGlobal: can only contain one?
                val tagOut: String
                val needGlobal: Boolean

                // first profile needs global, tag=proxy-global-x
                if (index == profileList.lastIndex) {
                    tagOut = "$TAG_AGENT-global-${proxyEntity.id}"
                    needGlobal = !pastExternal // why
                    bean.isFirstProfile = true
                } else {
                    // chain proxy:
                    // profile1 (in)  tag=proxy-global-x
                    // profile2       tag=proxy-x
                    // profile3 (out) tag=proxy
                    tagOut = if (index == 0) chainTag else {
                        "$chainTag-${proxyEntity.id}"
                    }
                    needGlobal = false
                }

                // profileList.lastIndex is first profile object in UI and it means "front proxy"
                // index == 0 means last profile in chain / not chain
                if (index == 0) {
                    chainOutboundTag = tagOut
                }

                if (needGlobal) {
                    if (globalOutbounds.contains(tagOut)) {
                        return@forEachIndexed
                    }
                    globalOutbounds.add(tagOut)
                }

                outboundTagsAll[tagOut] = proxyEntity

                if (index == 0) {
                    outboundTags.add(tagOut)
                    if (chainTag == TAG_AGENT) {
                        outboundTagsCurrent.add(tagOut)
                    }
                }

                // Chain outbound
                if (proxyEntity.needExternal()) {
                    val localPort = mkPort()
                    chainMap[localPort] = proxyEntity
                    currentOutbound.apply {
                        protocol = "socks"
                        settings = LazyOutboundConfigurationObject(
                            this, SocksOutboundConfigurationObject().apply {
                                servers = listOf(
                                    SocksOutboundConfigurationObject.ServerObject()
                                        .apply {
                                            address = LOCALHOST
                                            port = localPort
                                        })
                            })
                    }
                } else { // internal outbound
                    currentOutbound.apply {
                        val keepAliveInterval = DataStore.tcpKeepAliveInterval
                        val needKeepAliveInterval = keepAliveInterval !in intArrayOf(0, 15)
                        if (bean is StandardV2RayBean) {
                            when (bean) {
                                is SOCKSBean -> {
                                    protocol = "socks"
                                    settings = LazyOutboundConfigurationObject(
                                        this, SocksOutboundConfigurationObject().apply {
                                            servers = listOf(
                                                SocksOutboundConfigurationObject.ServerObject().apply {
                                                        address = bean.serverAddress
                                                        port = bean.serverPort
                                                        if (!bean.username.isNullOrBlank()) {
                                                            users = listOf(SocksOutboundConfigurationObject.ServerObject.UserObject()
                                                                .apply {
                                                                    user = bean.username
                                                                    pass = bean.password
                                                                })
                                                        }
                                                    })
                                            version = bean.protocolVersionName()
                                        })
                                }
                                is HttpBean -> {
                                    protocol = "http"
                                    settings = LazyOutboundConfigurationObject(
                                        this, HTTPOutboundConfigurationObject().apply {
                                            servers = listOf(
                                                HTTPOutboundConfigurationObject.ServerObject().apply {
                                                        address = bean.serverAddress
                                                        port = bean.serverPort
                                                        if (!bean.username.isNullOrBlank()) {
                                                            users = listOf(HTTPInboundConfigurationObject.AccountObject()
                                                                .apply {
                                                                    user = bean.username
                                                                    pass = bean.password
                                                                })
                                                        }
                                                    })
                                        })
                                }
                                is VMessBean -> {
                                    protocol = "vmess"
                                    settings = LazyOutboundConfigurationObject(
                                        this, VMessOutboundConfigurationObject().apply {
                                            vnext = listOf(
                                                VMessOutboundConfigurationObject.ServerObject().apply {
                                                        address = bean.serverAddress
                                                        port = bean.serverPort
                                                        users = listOf(VMessOutboundConfigurationObject.ServerObject.UserObject()
                                                            .apply {
                                                                id = bean.uuidOrGenerate()
                                                                alterId = bean.alterId
                                                                security = bean.encryption.takeIf { it.isNotBlank() }
                                                                    ?: "auto"
                                                                experimental = ""
                                                                if (bean.experimentalAuthenticatedLength) {
                                                                    experimental += "AuthenticatedLength"
                                                                }
                                                                if (bean.experimentalNoTerminationSignal) {
                                                                    experimental += "NoTerminationSignal"
                                                                }
                                                                if (experimental.isBlank()) experimental = null
                                                            })
                                                    when (bean.packetEncoding) {
                                                        1 -> {
                                                            packetEncoding = "packet"
                                                            if (currentDomainStrategy == "AsIs") {
                                                                currentDomainStrategy = "UseIP"
                                                            }
                                                        }
                                                        2 -> packetEncoding = "xudp"
                                                    }
                                                })
                                        })
                                }
                            }

                            streamSettings = StreamSettingsObject().apply {
                                network = bean.type
                                if (bean.security.isNotBlank()) {
                                    security = bean.security
                                }
                                if (security == "tls") {
                                    tlsSettings = TLSObject().apply {
                                        if (bean.sni.isNotBlank()) {
                                            serverName = bean.sni
                                        }

                                        if (bean.alpn.isNotBlank()) {
                                            alpn = bean.alpn.split("\n")
                                        }

                                        if (bean.certificates.isNotBlank()) {
                                            disableSystemRoot = true
                                            certificates = listOf(
                                                TLSObject.CertificateObject()
                                                    .apply {
                                                        usage = "verify"
                                                        certificate = bean.certificates.split(
                                                            "\n"
                                                        ).filter { it.isNotBlank() }
                                                    })
                                        }

                                        if (bean.pinnedPeerCertificateChainSha256.isNotBlank()) {
                                            pinnedPeerCertificateChainSha256 = bean.pinnedPeerCertificateChainSha256.split(
                                                "\n"
                                            ).filter { it.isNotBlank() }
                                        }

                                        if (bean.allowInsecure) {
                                            allowInsecure = true
                                        }
                                    }
                                }

                                when (network) {
                                    "tcp" -> {
                                        tcpSettings = TcpObject().apply {
                                            if (bean.headerType == "http") {
                                                header = TcpObject.HeaderObject().apply {
                                                    type = "http"
                                                    if (bean.host.isNotBlank() || bean.path.isNotBlank()) {
                                                        request = TcpObject.HeaderObject.HTTPRequestObject()
                                                            .apply {
                                                                headers = mutableMapOf()
                                                                if (bean.host.isNotBlank()) {
                                                                    headers["Host"] = TcpObject.HeaderObject.StringOrListObject()
                                                                        .apply {
                                                                            valueY = bean.host.split(
                                                                                ","
                                                                            ).map { it.trim() }
                                                                        }
                                                                }
                                                                if (bean.path.isNotBlank()) {
                                                                    path = bean.path.split(",")
                                                                }
                                                            }
                                                    }
                                                }
                                            }
                                        }
                                    }
                                    "kcp" -> {
                                        kcpSettings = KcpObject().apply {
                                            mtu = 1350
                                            tti = 50
                                            uplinkCapacity = 12
                                            downlinkCapacity = 100
                                            congestion = false
                                            readBufferSize = 1
                                            writeBufferSize = 1
                                            header = KcpObject.HeaderObject().apply {
                                                type = bean.headerType
                                            }
                                            if (bean.mKcpSeed.isNotBlank()) {
                                                seed = bean.mKcpSeed
                                            }
                                        }
                                    }
                                    "ws" -> {
                                        wsSettings = WebSocketObject().apply {
                                            headers = mutableMapOf()

                                            if (bean.host.isNotBlank()) {
                                                headers["Host"] = bean.host
                                            }

                                            path = bean.path.takeIf { it.isNotBlank() } ?: "/"

                                            if (bean.wsMaxEarlyData > 0) {
                                                maxEarlyData = bean.wsMaxEarlyData
                                            }

                                            if (bean.earlyDataHeaderName.isNotBlank()) {
                                                earlyDataHeaderName = bean.earlyDataHeaderName
                                            }

                                            if (bean.wsUseBrowserForwarder) {
                                                useBrowserForwarding = true
                                                requireWs = true
                                            }
                                        }
                                    }
                                    "http" -> {
                                        network = "http"

                                        httpSettings = HttpObject().apply {
                                            if (bean.host.isNotBlank()) {
                                                host = bean.host.split(",")
                                            }

                                            path = bean.path.takeIf { it.isNotBlank() } ?: "/"
                                        }
                                    }
                                    "quic" -> {
                                        quicSettings = QuicObject().apply {
                                            security = bean.quicSecurity.takeIf { it.isNotBlank() }
                                                ?: "none"
                                            key = bean.quicKey
                                            header = QuicObject.HeaderObject().apply {
                                                type = bean.headerType.takeIf { it.isNotBlank() }
                                                    ?: "none"
                                            }
                                        }
                                    }
                                    "grpc" -> {
                                        grpcSettings = GrpcObject().apply {
                                            serviceName = bean.grpcServiceName
                                        }
                                    }
                                }

                                if (needKeepAliveInterval) {
                                    sockopt = StreamSettingsObject.SockoptObject().apply {
                                        tcpKeepAliveInterval = keepAliveInterval
                                    }
                                }

                            }
                        } else if (bean is ShadowsocksBean || bean is ShadowsocksRBean) {
                            protocol = "shadowsocks"
                            settings = LazyOutboundConfigurationObject(
                                this,
                                ShadowsocksOutboundConfigurationObject().apply {
                                    servers = listOf(ShadowsocksOutboundConfigurationObject.ServerObject()
                                        .apply {
                                            address = bean.serverAddress
                                            port = bean.serverPort
                                            when (bean) {
                                                is ShadowsocksBean -> {
                                                    method = bean.method
                                                    password = bean.password
                                                }
                                                is ShadowsocksRBean -> {
                                                    method = bean.method
                                                    password = bean.password
                                                }
                                            }
                                        })
                                    if (needKeepAliveInterval) {
                                        streamSettings = StreamSettingsObject().apply {
                                            sockopt = StreamSettingsObject.SockoptObject().apply {
                                                tcpKeepAliveInterval = keepAliveInterval
                                            }
                                        }
                                    }
                                    if (bean is ShadowsocksRBean) {
                                        plugin = "shadowsocksr"
                                        pluginArgs = listOf(
                                            "--obfs=${bean.obfs}",
                                            "--obfs-param=${bean.obfsParam}",
                                            "--protocol=${bean.protocol}",
                                            "--protocol-param=${bean.protocolParam}"
                                        )
                                    } else if (bean is ShadowsocksBean && bean.plugin.isNotBlank()) {
                                        val pluginConfiguration = PluginConfiguration(bean.plugin)
                                        try {
                                            PluginManager.init(pluginConfiguration)
                                                ?.let { (path, opts, _) ->
                                                    plugin = path
                                                    pluginOpts = opts.toString()
                                                }
                                        } catch (e: PluginManager.PluginNotFoundException) {
                                            if (e.plugin in arrayOf("v2ray-plugin", "obfs-local")) {
                                                plugin = e.plugin
                                                pluginOpts = pluginConfiguration.getOptions()
                                                    .toString()
                                            } else {
                                                throw e
                                            }
                                        }
                                    }
                                })
                        } else if (bean is TrojanBean) {
                            protocol = "trojan"
                            settings = LazyOutboundConfigurationObject(
                                this,
                                TrojanOutboundConfigurationObject().apply {
                                    servers = listOf(
                                        TrojanOutboundConfigurationObject.ServerObject()
                                            .apply {
                                                address = bean.serverAddress
                                                port = bean.serverPort
                                                password = bean.password
                                            })
                                })
                            streamSettings = StreamSettingsObject().apply {
                                network = "tcp"
                                security = "tls"
                                tlsSettings = TLSObject().apply {
                                    if (bean.sni.isNotBlank()) {
                                        serverName = bean.sni
                                    }
                                    if (bean.alpn.isNotBlank()) {
                                        alpn = bean.alpn.split("\n")
                                    }
                                }
                                if (needKeepAliveInterval) {
                                    sockopt = StreamSettingsObject.SockoptObject().apply {
                                        tcpKeepAliveInterval = keepAliveInterval
                                    }
                                }
                                if (bean.allowInsecure) {
                                    tlsSettings = tlsSettings ?: TLSObject()
                                    tlsSettings.allowInsecure = true
                                }
                            }
                        } else if (bean is WireGuardBean) {
                            protocol = "wireguard"
                            settings = LazyOutboundConfigurationObject(
                                this,
                                WireGuardOutbounzConfigurationObject().apply {
                                    address = bean.finalAddress
                                    port = bean.finalPort
                                    network = "udp"
                                    localAddresses = bean.localAddress.split("\n")
                                    privateKey = bean.privateKey
                                    peerPublicKey = bean.peerPublicKey
                                    preSharedKey = bean.peerPreSharedKey
                                })
                            streamSettings = StreamSettingsObject().apply {
                                if (needKeepAliveInterval) {
                                    sockopt = StreamSettingsObject.SockoptObject().apply {
                                        tcpKeepAliveInterval = keepAliveInterval
                                    }
                                }
                            }
                        } else if (bean is SSHBean) {
                            protocol = "ssh"
                            settings = LazyOutboundConfigurationObject(
                                this,
                                SSHOutbountConfigurationObject().apply {
                                    address = bean.finalAddress
                                    port = bean.finalPort
                                    user = bean.username
                                    when (bean.authType) {
                                        SSHBean.AUTH_TYPE_PRIVATE_KEY -> {
                                            privateKey = bean.privateKey
                                            password = bean.privateKeyPassphrase
                                        }
                                        else -> {
                                            password = bean.password
                                        }
                                    }
                                    publicKey = bean.publicKey
                                })
                            streamSettings = StreamSettingsObject().apply {
                                if (needKeepAliveInterval) {
                                    sockopt = StreamSettingsObject.SockoptObject().apply {
                                        tcpKeepAliveInterval = keepAliveInterval
                                    }
                                }
                            }
                        }
                        if ((index == 0) && proxyEntity.needCoreMux() && DataStore.enableMux) {
                            mux = OutboundObject.MuxObject().apply {
                                enabled = true
                                concurrency = DataStore.muxConcurrency
                                if (bean is StandardV2RayBean) {
                                    when (bean.packetEncoding) {
                                        1 -> {
                                            packetEncoding = "packet"
                                        }
                                        2 -> {
                                            packetEncoding = "xudp"
                                        }
                                    }
                                }
                            }
                        }
                    }
                }

                currentOutbound.tag = tagOut
                currentOutbound.domainStrategy = currentDomainStrategy

                // chain rules
                if (index > 0) {
                    if (!pastExternal) {
                        pastOutbound.proxySettings = OutboundObject.ProxySettingsObject().apply {
                            tag = tagOut
                            transportLayer = true
                        }
                    } else {
                        routing.rules.add(RoutingObject.RuleObject().apply {
                            type = "field"
                            inboundTag = listOf(pastInboundTag)
                            outboundTag = tagOut
                        })
                    }
                }

                // External proxy need a dokodemo-door inbound to forward the traffic
                // For external proxy software, their traffic must goes to v2ray-core to use protected fd.
                if (bean.canMapping() && proxyEntity.needExternal()) {
                    val mappingPort = mkPort()
                    bean.finalAddress = LOCALHOST
                    bean.finalPort = mappingPort

                    inbounds.add(InboundObject().apply {
                        listen = LOCALHOST
                        port = mappingPort
                        tag = "$chainTag-mapping-${proxyEntity.id}"
                        protocol = "dokodemo-door"
                        settings = LazyInboundConfigurationObject(
                            this,
                            DokodemoDoorInboundConfigurationObject().apply {
                                address = bean.serverAddress
                                network = bean.network()
                                port = bean.serverPort
                            })

                        pastInboundTag = tag

                        // no chain rule and not outbound, so need to set to direct
                        if (bean.isFirstProfile) {
                            routing.rules.add(RoutingObject.RuleObject().apply {
                                type = "field"
                                inboundTag = listOf(tag)
                                outboundTag = TAG_DIRECT
                            })
                        }

                    })
                }

                outbounds.add(currentOutbound)
                chainOutbounds.add(currentOutbound)
                pastExternal = proxyEntity.needExternal()
                pastOutbound = currentOutbound

            }

            return chainOutboundTag

        }

        val tagProxy = buildChain(TAG_AGENT, proxies)
        val tagMap = mutableMapOf<Long, String>()
        extraProxies.forEach { (key, entities) ->
            tagMap[key] = buildChain("$TAG_AGENT-$key", entities)
        }

        val notVpn = DataStore.serviceMode != Key.MODE_VPN

        // apply user rules
        val uidListDNSRemote = mutableListOf<Int>()
        val uidListDNSDirect = mutableListOf<Int>()
        val domainListDNSRemote = mutableListOf<String>()
        val domainListDNSDirect = mutableListOf<String>()

        for (rule in extraRules) {
            val _uidList = rule.packages.map {
                PackageCache[it]?.takeIf { uid -> uid >= 10000 } ?: 1000
            }.toHashSet().toList()

            if (rule.packages.isNotEmpty()) {
                dumpUid = true
                if (notVpn) {
                    alerts.add(0 to rule.displayName())
                    continue
                }
            }
            routing.rules.add(RoutingObject.RuleObject().apply {
                type = "field"
                if (rule.packages.isNotEmpty()) {
                    PackageCache.awaitLoadSync()
                    uidList = _uidList
                }

                var _domainList: List<String>? = null
                if (rule.domains.isNotBlank()) {
                    domain = rule.domains.split("\n")
                    _domainList = domain
                }
                if (rule.ip.isNotBlank()) {
                    ip = rule.ip.split("\n")
                }
                if (rule.port.isNotBlank()) {
                    port = rule.port
                }
                if (rule.sourcePort.isNotBlank()) {
                    sourcePort = rule.sourcePort
                }
                if (rule.network.isNotBlank()) {
                    network = rule.network
                }
                if (rule.source.isNotBlank()) {
                    source = rule.source.split("\n")
                }
                if (rule.protocol.isNotBlank()) {
                    protocol = rule.protocol.split("\n")
                }
                if (rule.attrs.isNotBlank()) {
                    attrs = rule.attrs
                }

                if (rule.reverse) inboundTag = listOf("reverse-${rule.id}")

                // also bypass lookup
                // cannot use other outbound profile to lookup...
                if (rule.outbound == -1L) {
                    uidListDNSDirect += _uidList
                    if (_domainList != null) domainListDNSDirect += _domainList
                } else if (rule.outbound == 0L) {
                    uidListDNSRemote += _uidList
                    if (_domainList != null) domainListDNSRemote += _domainList
                }

                outboundTag = when (val outId = rule.outbound) {
                    0L -> tagProxy
                    -1L -> TAG_BYPASS
                    -2L -> TAG_BLOCK
                    else -> if (outId == proxy.id) tagProxy else tagMap[outId]
                }
            })

            if (rule.reverse) {
                outbounds.add(OutboundObject().apply {
                    tag = "reverse-out-${rule.id}"
                    protocol = "freedom"
                    settings = LazyOutboundConfigurationObject(
                        this,
                        FreedomOutboundConfigurationObject().apply {
                            redirect = rule.redirect
                        })
                })
                if (reverse == null) {
                    reverse = ReverseObject().apply {
                        bridges = ArrayList()
                    }
                }
                reverse.bridges.add(ReverseObject.BridgeObject().apply {
                    tag = "reverse-${rule.id}"
                    domain = rule.domains.substringAfter("full:")
                })
                routing.rules.add(RoutingObject.RuleObject().apply {
                    type = "field"
                    inboundTag = listOf("reverse-${rule.id}")
                    outboundTag = "reverse-out-${rule.id}"
                })
            }

        }

        if (requireWs) {
            browserForwarder = BrowserForwarderObject().apply {
                listenAddr = LOCALHOST
                listenPort = mkPort()
            }
        }

        for (freedom in arrayOf(TAG_DIRECT, TAG_BYPASS)) outbounds.add(OutboundObject().apply {
            tag = freedom
            protocol = "freedom"
        })

        outbounds.add(OutboundObject().apply {
            tag = TAG_BLOCK
            protocol = "blackhole"
            /* settings = LazyOutboundConfigurationObject(this,
                 BlackholeOutboundConfigurationObject().apply {
                     keepConnection = true
                 })*/
        })

        if (!forTest) {
            inbounds.add(InboundObject().apply {
                tag = TAG_DNS_IN
                listen = bind
                port = DataStore.localDNSPort
                protocol = "dokodemo-door"
                settings = LazyInboundConfigurationObject(
                    this,
                    DokodemoDoorInboundConfigurationObject().apply {
                        address = if (!remoteDns.first().isIpAddress()) {
                            "1.0.0.1"
                        } else {
                            remoteDns.first()
                        }
                        network = "tcp,udp"
                        port = 53
                    })

            })

            outbounds.add(OutboundObject().apply {
                protocol = "dns"
                tag = TAG_DNS_OUT
                settings = LazyOutboundConfigurationObject(
                    this,
                    DNSOutboundConfigurationObject().apply {
                        userLevel = 1
                        var dns = remoteDns.first()
                        if (dns.contains(":")) {
                            val lPort = dns.substringAfterLast(":")
                            dns = dns.substringBeforeLast(":")
                            if (NumberUtil.isInteger(lPort)) {
                                port = lPort.toInt()
                            }
                        }
                        if (dns.isIpAddress()) {
                            address = dns
                        } else if (dns.contains("://")) {
                            network = "tcp"
                            address = dns.substringAfter("://")
                        }
                    })
            })
        }

        if (DataStore.directDnsUseSystem) {
            // finally able to use "localDns" now...
            directDNS = listOf("localhost")
        }

        // routing for DNS server
        for (dns in remoteDns) {
            if (!dns.isIpAddress()) continue
            routing.rules.add(0, RoutingObject.RuleObject().apply {
                type = "field"
                outboundTag = tagProxy
                ip = listOf(dns)
            })
        }

        for (dns in directDNS) {
            if (!dns.isIpAddress()) continue

            routing.rules.add(0, RoutingObject.RuleObject().apply {
                type = "field"
                outboundTag = TAG_DIRECT
                ip = listOf(dns)
            })
        }

        // No need to "bypass IP"
        // see buildChain()
        val directLookupDomain = HashSet<String>()

        // Bypass Lookup for the first profile
        (proxies + extraProxies.values.flatten()).filter { it.requireBean().isFirstProfile }.forEach {
            it.requireBean().apply {
                if (!serverAddress.isIpAddress()) {
                    directLookupDomain.add("full:$serverAddress")
                }
            }
        }

        remoteDns.forEach {
            var address = it
            if (address.contains("://")) {
                address = address.substringAfter("://")
            }
            "https://$address".toHttpUrlOrNull()?.apply {
                if (!host.isIpAddress()) {
                    directLookupDomain.add("full:$host")
                }
            }
        }

        // dns object user rules
        // Note: "geosite:cn" matches before user rule... v2ray-core
        if (enableDnsRouting) {
            dns.servers[0].valueY?.uidList = uidListDNSRemote.toHashSet().toList()
            dns.servers[0].valueY?.domains = domainListDNSRemote.toHashSet().toList()
            directLookupDomain += domainListDNSDirect
        }

        // add directDNS objects here
        dns.servers.addAll(directDNS.map {
            DnsObject.StringOrServerObject().apply {
                valueY = DnsObject.ServerObject().apply {
                    address = it
                    domains = directLookupDomain.toList()
                    skipFallback = true
                    uidList = uidListDNSDirect.toHashSet().toList()
                }
            }
        })

        if (useFakeDns) {
            dns.servers.add(0, DnsObject.StringOrServerObject().apply {
                valueX = "fakedns"
            })
        }

        if (!forTest) routing.rules.add(0, RoutingObject.RuleObject().apply {
            type = "field"
            inboundTag = listOf(TAG_DNS_IN)
            outboundTag = TAG_DNS_OUT
        })

        if (allowAccess) {
            // temp: fix crash
            routing.rules.add(RoutingObject.RuleObject().apply {
                type = "field"
                ip = listOf("255.255.255.255")
                outboundTag = TAG_BLOCK
            })
        }

        if (trafficStatistics) stats = emptyMap()
    }.let {
        V2rayBuildResult(
            gson.toJson(it),
            indexMap,
            requireWs,
            if (requireWs) it.browserForwarder.listenPort else 0,
            outboundTags,
            outboundTagsCurrent,
            outboundTagsAll,
            TAG_BYPASS,
            it.observatory?.subjectSelector ?: HashSet(),
            dumpUid,
            alerts
        )
    }

}
