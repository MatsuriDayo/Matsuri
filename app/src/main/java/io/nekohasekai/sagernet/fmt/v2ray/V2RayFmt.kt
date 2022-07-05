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

package io.nekohasekai.sagernet.fmt.v2ray

import io.nekohasekai.sagernet.fmt.trojan.TrojanBean
import io.nekohasekai.sagernet.ktx.*
import moe.matsuri.nya.utils.NGUtil
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrl
import org.json.JSONObject

fun parseV2Ray(link: String): StandardV2RayBean {
    // Try parse stupid formats first

    if (!link.contains("?")) {
        try {
            return parseV2RayN(link)
        } catch (e: Exception) {
            Logs.i("try v2rayN: " + e.readableMessage)
        }
    }

    try {
        return tryResolveVmess4Kitsunebi(link)
    } catch (e: Exception) {
        Logs.i("try Kitsunebi: " + e.readableMessage)
    }

    // SagerNet: parse standard format

    val bean = VMessBean()
    val url = link.replace("vmess://", "https://").toHttpUrl()

    if (url.password.isNotBlank()) { // https://github.com/v2fly/v2fly-github-io/issues/26
        bean.serverAddress = url.host
        bean.serverPort = url.port
        bean.name = url.fragment

        var protocol = url.username
        bean.type = protocol
        bean.alterId = url.password.substringAfterLast('-').toInt()
        bean.uuid = url.password.substringBeforeLast('-')

        if (protocol.endsWith("+tls")) {
            bean.security = "tls"
            protocol = protocol.substring(0, protocol.length - 4)

            url.queryParameter("tlsServerName")?.let {
                if (it.isNotBlank()) {
                    bean.sni = it
                }
            }
        }

        when (protocol) {
            "tcp" -> {
                url.queryParameter("type")?.let { type ->
                    if (type == "http") {
                        bean.headerType = "http"
                        url.queryParameter("host")?.let {
                            bean.host = it
                        }
                    }
                }
            }
            "http" -> {
                url.queryParameter("path")?.let {
                    bean.path = it
                }
                url.queryParameter("host")?.let {
                    bean.host = it.split("|").joinToString(",")
                }
            }
            "ws" -> {
                url.queryParameter("path")?.let {
                    bean.path = it
                }
                url.queryParameter("host")?.let {
                    bean.host = it
                }
            }
            "kcp" -> {
                url.queryParameter("type")?.let {
                    bean.headerType = it
                }
                url.queryParameter("seed")?.let {
                    bean.mKcpSeed = it
                }
            }
            "quic" -> {
                url.queryParameter("security")?.let {
                    bean.quicSecurity = it
                }
                url.queryParameter("key")?.let {
                    bean.quicKey = it
                }
                url.queryParameter("type")?.let {
                    bean.headerType = it
                }
            }
        }
    } else {
        bean.parseDuckSoft(url)
    }

    Logs.d(formatObject(bean))

    return bean
}

// https://github.com/XTLS/Xray-core/issues/91
// NO allowInsecure
fun StandardV2RayBean.parseDuckSoft(url: HttpUrl) {
    serverAddress = url.host
    serverPort = url.port
    name = url.fragment

    if (this is TrojanBean) {
        password = url.username
    } else {
        uuid = url.username
    }

    if (url.pathSegments.size > 1 || url.pathSegments[0].isNotBlank()) {
        path = url.pathSegments.joinToString("/")
    }

    type = url.queryParameter("type") ?: "tcp"
    security = url.queryParameter("security")
    if (security == null) {
        security = if (this is TrojanBean) {
            "tls"
        } else {
            "none"
        }
    }

    when (security) {
        "tls" -> {
            url.queryParameter("sni")?.let {
                sni = it
            }
            url.queryParameter("alpn")?.let {
                alpn = it
            }
            url.queryParameter("cert")?.let {
                certificates = it
            }
            url.queryParameter("chain")?.let {
                pinnedPeerCertificateChainSha256 = it
            }
        }
    }
    when (type) {
        "tcp" -> {
            url.queryParameter("headerType")?.let { ht ->
                if (ht == "http") {
                    headerType = ht
                    url.queryParameter("host")?.let {
                        host = it
                    }
                    url.queryParameter("path")?.let {
                        path = it
                    }
                }
            }
        }
        "kcp" -> {
            url.queryParameter("headerType")?.let {
                headerType = it
            }
            url.queryParameter("seed")?.let {
                mKcpSeed = it
            }
        }
        "http" -> {
            url.queryParameter("host")?.let {
                host = it
            }
            url.queryParameter("path")?.let {
                path = it
            }
        }
        "ws" -> {
            url.queryParameter("host")?.let {
                host = it
            }
            url.queryParameter("path")?.let {
                path = it
            }
            url.queryParameter("ed")?.let { ed ->
                wsMaxEarlyData = ed.toInt()

                url.queryParameter("eh")?.let {
                    earlyDataHeaderName = it
                }
            }
        }
        "quic" -> {
            url.queryParameter("headerType")?.let {
                headerType = it
            }
            url.queryParameter("quicSecurity")?.let { qs ->
                quicSecurity = qs
                url.queryParameter("key")?.let {
                    quicKey = it
                }
            }
        }
        "grpc" -> {
            url.queryParameter("serviceName")?.let {
                grpcServiceName = it
            }
        }
    }

    url.queryParameter("packetEncoding")?.let {
        when (it) {
            "packet" -> packetEncoding = 1
            "xudp" -> packetEncoding = 2
        }
    }
}

// 不确定是谁的格式
private fun tryResolveVmess4Kitsunebi(server: String): VMessBean {
    // vmess://YXV0bzo1YWY1ZDBlYy02ZWEwLTNjNDMtOTNkYi1jYTMwMDg1MDNiZGJAMTgzLjIzMi41Ni4xNjE6MTIwMg
    // ?remarks=*%F0%9F%87%AF%F0%9F%87%B5JP%20-355%20TG@moon365free&obfsParam=%7B%22Host%22:%22183.232.56.161%22%7D&path=/v2ray&obfs=websocket&alterId=0

    var result = server.replace("vmess://", "")
    val indexSplit = result.indexOf("?")
    if (indexSplit > 0) {
        result = result.substring(0, indexSplit)
    }
    result = NGUtil.decode(result)

    val arr1 = result.split('@')
    if (arr1.count() != 2) {
        throw IllegalStateException("invalid kitsunebi format")
    }
    val arr21 = arr1[0].split(':')
    val arr22 = arr1[1].split(':')
    if (arr21.count() != 2) {
        throw IllegalStateException("invalid kitsunebi format")
    }

    return VMessBean().apply {
        serverAddress = arr22[0]
        serverPort = NGUtil.parseInt(arr22[1])
        uuid = arr21[1]
        encryption = arr21[0]
        if (indexSplit < 0) return@apply

        val url = ("https://localhost/path?" + server.substringAfter("?")).toHttpUrl()
        url.queryParameter("remarks")?.apply { name = this }
        url.queryParameter("alterId")?.apply { alterId = this.toInt() }
        url.queryParameter("path")?.apply { path = this }
        url.queryParameter("tls")?.apply { security = "tls" }
        url.queryParameter("allowInsecure")
            ?.apply { if (this == "1" || this == "true") allowInsecure = true }
        url.queryParameter("obfs")?.apply {
            type = this.replace("websocket", "ws").replace("none", "tcp")
            if (type == "ws") {
                url.queryParameter("obfsParam")?.apply {
                    if (this.startsWith("{")) {
                        host = JSONObject(this).getStr("Host")
                    } else if (security == "tls") {
                        sni = this
                    }
                }
            }
        }
    }
}

// SagerNet's
// Do not support some format and then throw exception
fun parseV2RayN(link: String): VMessBean {
    val result = link.substringAfter("vmess://").decodeBase64UrlSafe()
    if (result.contains("= vmess")) {
        return parseCsvVMess(result)
    }
    val bean = VMessBean()
    val json = JSONObject(result)

    bean.serverAddress = json.getStr("add") ?: ""
    bean.serverPort = json.getIntNya("port") ?: 1080
    bean.encryption = json.getStr("scy") ?: ""
    bean.uuid = json.getStr("id") ?: ""
    bean.alterId = json.getIntNya("aid") ?: 0
    bean.type = json.getStr("net") ?: ""
    bean.headerType = json.getStr("type") ?: ""
    bean.host = json.getStr("host") ?: ""
    bean.path = json.getStr("path") ?: ""

    when (bean.type) {
        "quic" -> {
            bean.quicSecurity = bean.host
            bean.quicKey = bean.path
        }
        "kcp" -> {
            bean.mKcpSeed = bean.path
        }
        "grpc" -> {
            bean.grpcServiceName = bean.path
        }
    }

    bean.name = json.getStr("ps") ?: ""
    bean.sni = json.getStr("sni") ?: bean.host
    bean.security = json.getStr("tls")

    if (json.optInt("v", 2) < 2) {
        when (bean.type) {
            "ws" -> {
                var path = ""
                var host = ""
                val lstParameter = bean.host.split(";")
                if (lstParameter.isNotEmpty()) {
                    path = lstParameter[0].trim()
                }
                if (lstParameter.size > 1) {
                    path = lstParameter[0].trim()
                    host = lstParameter[1].trim()
                }
                bean.path = path
                bean.host = host
            }
            "h2" -> {
                var path = ""
                var host = ""
                val lstParameter = bean.host.split(";")
                if (lstParameter.isNotEmpty()) {
                    path = lstParameter[0].trim()
                }
                if (lstParameter.size > 1) {
                    path = lstParameter[0].trim()
                    host = lstParameter[1].trim()
                }
                bean.path = path
                bean.host = host
            }
        }
    }

    return bean

}

private fun parseCsvVMess(csv: String): VMessBean {

    val args = csv.split(",")

    val bean = VMessBean()

    bean.serverAddress = args[1]
    bean.serverPort = args[2].toInt()
    bean.encryption = args[3]
    bean.uuid = args[4].replace("\"", "")

    args.subList(5, args.size).forEach {

        when {
            it == "over-tls=true" -> bean.security = "tls"
            it.startsWith("tls-host=") -> bean.host = it.substringAfter("=")
            it.startsWith("obfs=") -> bean.type = it.substringAfter("=")
            it.startsWith("obfs-path=") || it.contains("Host:") -> {
                runCatching {
                    bean.path = it.substringAfter("obfs-path=\"").substringBefore("\"obfs")
                }
                runCatching {
                    bean.host = it.substringAfter("Host:").substringBefore("[")
                }

            }

        }

    }

    return bean

}

fun VMessBean.toV2rayN(): String {

    return "vmess://" + JSONObject().apply {

        put("v", 2)
        put("ps", name)
        put("add", serverAddress)
        put("port", serverPort)
        put("id", uuid)
        put("aid", alterId)
        put("net", type)
        put("host", host)
        put("path", path)
        put("type", headerType)

        when (headerType) {
            "quic" -> {
                put("host", quicSecurity)
                put("path", quicKey)
            }
            "kcp" -> {
                put("path", mKcpSeed)
            }
            "grpc" -> {
                put("path", grpcServiceName)
            }
        }

        put("tls", if (security == "tls") "tls" else "")
        put("sni", sni)
        put("scy", encryption)

    }.toStringPretty().let { NGUtil.encode(it) }

}

fun StandardV2RayBean.toUri(standard: Boolean = true): String {
    if (this is VMessBean && alterId > 0) return toV2rayN()

    val builder = linkBuilder().username(if (this is TrojanBean) password else uuid)
        .host(serverAddress)
        .port(serverPort)
        .addQueryParameter("type", type)
    if (this !is TrojanBean) builder.addQueryParameter("encryption", encryption)

    when (type) {
        "tcp" -> {
            if (headerType == "http") {
                builder.addQueryParameter("headerType", headerType)

                if (host.isNotBlank()) {
                    builder.addQueryParameter("host", host)
                }
                if (path.isNotBlank()) {
                    if (standard) {
                        builder.addQueryParameter("path", path)
                    } else {
                        builder.encodedPath(path.pathSafe())
                    }
                }
            }
        }
        "kcp" -> {
            if (headerType.isNotBlank() && headerType != "none") {
                builder.addQueryParameter("headerType", headerType)
            }
            if (mKcpSeed.isNotBlank()) {
                builder.addQueryParameter("seed", mKcpSeed)
            }
        }
        "ws", "http" -> {
            if (host.isNotBlank()) {
                builder.addQueryParameter("host", host)
            }
            if (path.isNotBlank()) {
                if (standard) {
                    builder.addQueryParameter("path", path)
                } else {
                    builder.encodedPath(path.pathSafe())
                }
            }
            if (type == "ws") {
                if (wsMaxEarlyData > 0) {
                    builder.addQueryParameter("ed", "$wsMaxEarlyData")
                    if (earlyDataHeaderName.isNotBlank()) {
                        builder.addQueryParameter("eh", earlyDataHeaderName)
                    }
                }
            }
        }
        "quic" -> {
            if (headerType.isNotBlank() && headerType != "none") {
                builder.addQueryParameter("headerType", headerType)
            }
            if (quicSecurity.isNotBlank() && quicSecurity != "none") {
                builder.addQueryParameter("quicSecurity", quicSecurity)
                builder.addQueryParameter("key", quicKey)
            }
        }
        "grpc" -> {
            if (grpcServiceName.isNotBlank()) {
                builder.addQueryParameter("serviceName", grpcServiceName)
            }
        }
    }

    if (security.isNotBlank() && security != "none") {
        builder.addQueryParameter("security", security)
        when (security) {
            "tls" -> {
                if (sni.isNotBlank()) {
                    builder.addQueryParameter("sni", sni)
                }
                if (alpn.isNotBlank()) {
                    builder.addQueryParameter("alpn", alpn)
                }
                if (certificates.isNotBlank()) {
                    builder.addQueryParameter("cert", certificates)
                }
                if (pinnedPeerCertificateChainSha256.isNotBlank()) {
                    builder.addQueryParameter("chain", pinnedPeerCertificateChainSha256)
                }
                if (allowInsecure) builder.addQueryParameter("allowInsecure", "1")
            }
        }
    }

    when (packetEncoding) {
        1 -> {
            builder.addQueryParameter("packetEncoding", "packet")
        }
        2 -> {
            builder.addQueryParameter("packetEncoding", "xudp")
        }
    }

    if (name.isNotBlank()) {
        builder.encodedFragment(name.urlSafe())
    }

    return builder.toLink("vmess")

}