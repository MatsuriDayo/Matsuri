package moe.matsuri.nya

import io.nekohasekai.sagernet.database.DataStore
import moe.matsuri.nya.neko.NekoPluginManager

// Settings for all protocols, built-in or plugin
object Protocols {
    fun shouldEnableMux(protocol: String): Boolean {
        return DataStore.muxProtocols.contains(protocol)
    }

    fun getCanMuxList(): List<String> {
        // built-in and support mux
        val list = mutableListOf("vmess", "trojan", "trojan-go")

        NekoPluginManager.getProtocols().forEach {
            if (it.protocolConfig.optBoolean("canMux")) {
                list.add(it.protocolId)
            }
        }

        return list
    }
}