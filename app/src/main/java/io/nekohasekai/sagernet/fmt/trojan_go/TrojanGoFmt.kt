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

package io.nekohasekai.sagernet.fmt.trojan_go

import com.github.shadowsocks.plugin.PluginConfiguration
import com.github.shadowsocks.plugin.PluginManager
import com.github.shadowsocks.plugin.PluginOptions
import io.nekohasekai.sagernet.IPv6Mode
import io.nekohasekai.sagernet.database.DataStore
import io.nekohasekai.sagernet.fmt.LOCALHOST
import io.nekohasekai.sagernet.fmt.shadowsocks.fixInvalidParams
import io.nekohasekai.sagernet.ktx.*
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import org.json.JSONArray
import org.json.JSONObject

fun parseTrojanGo(server: String): TrojanGoBean {
    val link = server.replace("trojan-go://", "https://").toHttpUrlOrNull() ?: error(
        "invalid trojan-link link $server"
    )
    return TrojanGoBean().apply {
        serverAddress = link.host
        serverPort = link.port
        password = link.username
        link.queryParameter("sni")?.let {
            sni = it
        }
        link.queryParameter("type")?.let { lType ->
            type = lType

            when (type) {
                "ws" -> {
                    link.queryParameter("host")?.let {
                        host = it
                    }
                    link.queryParameter("path")?.let {
                        path = it
                    }
                }
                else -> {
                }
            }
        }
        link.queryParameter("encryption")?.let {
            encryption = it
        }
        link.queryParameter("plugin")?.let {
            plugin = it
        }
        link.fragment.takeIf { !it.isNullOrBlank() }?.let {
            name = it
        }
    }
}

fun TrojanGoBean.toUri(): String {
    val builder = linkBuilder().username(password).host(serverAddress).port(serverPort)
    if (sni.isNotBlank()) {
        builder.addQueryParameter("sni", sni)
    }
    if (type.isNotBlank() && type != "original") {
        builder.addQueryParameter("type", type)

        when (type) {
            "ws" -> {
                if (host.isNotBlank()) {
                    builder.addQueryParameter("host", host)
                }
                if (path.isNotBlank()) {
                    builder.addQueryParameter("path", path)
                }
            }
        }
    }
    if (type.isNotBlank() && type != "none") {
        builder.addQueryParameter("encryption", encryption)
    }
    if (plugin.isNotBlank()) {
        builder.addQueryParameter("plugin", plugin)
    }

    if (name.isNotBlank()) {
        builder.encodedFragment(name.urlSafe())
    }

    return builder.toLink("trojan-go")
}

fun TrojanGoBean.buildTrojanGoConfig(port: Int, mux: Boolean): String {
    return JSONObject().apply {
        put("run_type", "client")
        put("local_addr", LOCALHOST)
        put("local_port", port)
        put("remote_addr", finalAddress)
        put("remote_port", finalPort)
        put("password", JSONArray().apply {
            put(password)
        })
        put("log_level", if (DataStore.enableLog) 0 else 2)
        if (mux) put("mux", JSONObject().apply {
            put("enabled", true)
            put("concurrency", DataStore.muxConcurrency)
        })
        put("tcp", JSONObject().apply {
            put("prefer_ipv4", DataStore.ipv6Mode <= IPv6Mode.ENABLE)
        })

        when (type) {
            "original" -> {
            }
            "ws" -> put("websocket", JSONObject().apply {
                put("enabled", true)
                put("host", host)
                put("path", path)
            })
        }

        if (sni.isBlank() && finalAddress == LOCALHOST && !serverAddress.isIpAddress()) {
            sni = serverAddress
        }

        put("ssl", JSONObject().apply {
            if (sni.isNotBlank()) put("sni", sni)
        })

        when {
            encryption == "none" -> {
            }
            encryption.startsWith("ss;") -> put("shadowsocks", JSONObject().apply {
                put("enabled", true)
                put("method", encryption.substringAfter(";").substringBefore(":"))
                put("password", encryption.substringAfter(":"))
            })
        }

        if (plugin.isNotBlank()) {
            val pluginConfiguration = PluginConfiguration(plugin ?: "")
            PluginManager.init(pluginConfiguration)?.let { (path, opts, isV2) ->
                put("transport_plugin", JSONObject().apply {
                    put("enabled", true)
                    put("type", "shadowsocks")
                    put("command", path)
                    put("option", opts.toString())
                })
            }
        }
    }.toStringPretty()
}

fun JSONObject.parseTrojanGo(): TrojanGoBean {
    return TrojanGoBean().applyDefaultValues().apply {
        serverAddress = optString("remote_addr", serverAddress)
        serverPort = optInt("remote_port", serverPort)
        when (val pass = get("password")) {
            is String -> {
                password = pass
            }
            is List<*> -> {
                password = pass[0] as String
            }
        }
        optJSONArray("ssl")?.apply {
            sni = optString("sni", sni)
        }
        optJSONArray("websocket")?.apply {
            if (optBoolean("enabled", false)) {
                type = "ws"
                host = optString("host", host)
                path = optString("path", path)
            }
        }
        optJSONArray("shadowsocks")?.apply {
            if (optBoolean("enabled", false)) {
                encryption = "ss;${optString("method", "")}:${optString("password", "")}"
            }
        }
        optJSONArray("transport_plugin")?.apply {
            if (optBoolean("enabled", false)) {
                when (type) {
                    "shadowsocks" -> {
                        val pl = PluginConfiguration()
                        pl.selected = optString("command")
                        optJSONArray("arg")?.also {
                            pl.pluginsOptions[pl.selected] = PluginOptions().also { opts ->
                                var key = ""
                                for (index in 0 until it.length()) {
                                    if (index % 2 != 0) {
                                        key = it[index].toString()
                                    } else {
                                        opts[key] = it[index].toString()
                                    }
                                }
                            }
                        }
                        optString("option").also {
                            if (it != "") pl.pluginsOptions[pl.selected] = PluginOptions(it)
                        }
                        pl.fixInvalidParams()
                        plugin = pl.toString()
                    }
                }
            }
        }
    }
}