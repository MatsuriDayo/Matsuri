package io.nekohasekai.sagernet.utils

import android.content.Context
import android.content.Context.CONNECTIVITY_SERVICE
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.os.Build
import android.util.Log
import cn.hutool.core.lang.Validator
import io.nekohasekai.sagernet.SagerNet
import io.nekohasekai.sagernet.database.DataStore

import java.net.InetAddress

object SystemDNS {

    private fun get(): List<InetAddress> {
        val network = if (Build.VERSION.SDK_INT >= 23) {
            SagerNet.connectivity.activeNetwork
        } else {
            SagerNet.connectivity.allNetworks.find {
                SagerNet.connectivity.getNetworkInfo(it)?.isConnected == true
            }
        } ?: return emptyList()

        val linkProperties = SagerNet.connectivity.getLinkProperties(network) ?: return emptyList()

        // TODO filter VPN
        if (SagerNet.connectivity.getNetworkCapabilities(network)
                ?.hasTransport(NetworkCapabilities.TRANSPORT_VPN) == true
        ) {
            return emptyList()
        }

        return linkProperties.dnsServers
    }

    fun prepareSystemDNS() {
        try {
            val m = arrayListOf<String>()
            get().forEach {
                it.hostAddress?.apply {
                    if (Validator.isIpv4(this)) m.add(this)
                }
            }
            DataStore.directDnsSystem = m.joinToString("\n")
        } catch (e: Exception) {
            Log.w("nya", e.stackTraceToString())
        }
    }
}