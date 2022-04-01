package moe.matsuri.nya

import android.content.Context
import io.nekohasekai.sagernet.R
import io.nekohasekai.sagernet.database.DataStore
import io.nekohasekai.sagernet.database.ProxyEntity.Companion.TYPE_NEKO
import io.nekohasekai.sagernet.fmt.AbstractBean
import io.nekohasekai.sagernet.ktx.getColorAttr
import moe.matsuri.nya.neko.NekoPluginManager

// Settings for all protocols, built-in or plugin
object Protocols {
    // Mux

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

    // Deduplication

    class Deduplication(
        val bean: AbstractBean
    ) {

        fun hash(): String {
            return bean.serverAddress + bean.serverPort
        }

        override fun hashCode(): Int {
            return hash().toByteArray().contentHashCode()
        }

        override fun equals(other: Any?): Boolean {
            if (this === other) return true
            if (javaClass != other?.javaClass) return false

            other as Deduplication

            return hash() == other.hash()
        }

    }

    // Display

    fun Context.getProtocolColor(type: Int): Int {
        return when (type) {
            TYPE_NEKO -> getColorAttr(android.R.attr.textColorPrimary)
            else -> getColorAttr(R.attr.accentOrTextSecondary)
        }
    }

}