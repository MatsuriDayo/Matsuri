package io.nekohasekai.sagernet.fmt.socks

import io.nekohasekai.sagernet.ktx.decodeBase64UrlSafe
import io.nekohasekai.sagernet.ktx.isTLS
import io.nekohasekai.sagernet.ktx.toLink
import io.nekohasekai.sagernet.ktx.urlSafe
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

fun parseSOCKS(link: String): SOCKSBean {
    val url = ("http://" + link.substringAfter("://")).toHttpUrlOrNull()
        ?: error("Not supported: $link")

    return SOCKSBean().apply {
        protocol = when {
            link.startsWith("socks4://") -> SOCKSBean.PROTOCOL_SOCKS4
            link.startsWith("socks4a://") -> SOCKSBean.PROTOCOL_SOCKS4A
            else -> SOCKSBean.PROTOCOL_SOCKS5
        }
        serverAddress = url.host
        serverPort = url.port
        username = url.username
        password = url.password
        // v2rayN fmt
        if (password.isNullOrBlank() && !username.isNullOrBlank()) {
            try {
                val n = username.decodeBase64UrlSafe()
                username = n.substringBefore(":")
                password = n.substringAfter(":")
            } catch (_: Exception) {
            }
        }
    }
}

fun SOCKSBean.toUri(): String {

    val builder = HttpUrl.Builder().scheme("http").host(serverAddress).port(serverPort)
    if (!username.isNullOrBlank()) builder.username(username)
    if (!password.isNullOrBlank()) builder.password(password)
    if (isTLS()) {
        builder.addQueryParameter("tls", "true")
        if (sni.isNotBlank()) {
            builder.addQueryParameter("sni", sni)
        }
    }
    if (!name.isNullOrBlank()) builder.encodedFragment(name.urlSafe())
    return builder.toLink("socks${protocolVersion()}")

}
