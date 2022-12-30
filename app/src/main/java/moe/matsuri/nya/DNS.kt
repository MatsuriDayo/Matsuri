package moe.matsuri.nya

import io.nekohasekai.sagernet.database.DataStore
import io.nekohasekai.sagernet.fmt.v2ray.V2RayConfig.DnsObject.ServerObject

object DNS {
    fun ServerObject.applyDNSNetworkSettings(isDirect: Boolean) {
        if (isDirect) {
            if (DataStore.dnsNetwork.contains("NoDirectIPv4")) this.queryStrategy = "UseIPv6"
            if (DataStore.dnsNetwork.contains("NoDirectIPv6")) this.queryStrategy = "UseIPv4"
        } else {
            if (DataStore.dnsNetwork.contains("NoRemoteIPv4")) this.queryStrategy = "UseIPv6"
            if (DataStore.dnsNetwork.contains("NoRemoteIPv6")) this.queryStrategy = "UseIPv4"
        }
    }
}
