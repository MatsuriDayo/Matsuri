/******************************************************************************
 *                                                                            *
 * Copyright (C) 2021 by nekohasekai <contact-sagernet@sekai.icu>             *
 * Copyright (C) 2021 by Max Lv <max.c.lv@gmail.com>                          *
 * Copyright (C) 2021 by Mygod Studio <contact-shadowsocks-android@mygod.be>  *
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

package io.nekohasekai.sagernet.bg

import android.Manifest
import android.annotation.SuppressLint
import android.app.Service
import android.content.Intent
import android.content.pm.PackageManager
import android.net.DnsResolver
import android.net.LocalSocket
import android.net.Network
import android.net.ProxyInfo
import android.os.Build
import android.os.CancellationSignal
import android.os.ParcelFileDescriptor
import android.os.PowerManager
import android.system.ErrnoException
import android.system.Os
import androidx.annotation.RequiresApi
import com.github.shadowsocks.net.ConcurrentLocalSocketListener
import io.nekohasekai.sagernet.*
import io.nekohasekai.sagernet.database.DataStore
import io.nekohasekai.sagernet.database.SagerDatabase
import io.nekohasekai.sagernet.database.StatsEntity
import io.nekohasekai.sagernet.fmt.LOCALHOST
import io.nekohasekai.sagernet.fmt.hysteria.HysteriaBean
import io.nekohasekai.sagernet.ktx.*
import io.nekohasekai.sagernet.ui.VpnRequestActivity
import io.nekohasekai.sagernet.utils.PackageCache
import io.nekohasekai.sagernet.utils.Subnet
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.asExecutor
import kotlinx.coroutines.runBlocking
import libcore.*
import moe.matsuri.nya.neko.needBypassRootUid
import java.io.File
import java.io.FileDescriptor
import java.io.IOException
import java.net.InetAddress
import kotlin.coroutines.suspendCoroutine
import android.net.VpnService as BaseVpnService

class VpnService : BaseVpnService(),
    BaseService.Interface,
    TrafficListener,
    LocalResolver,
    Protector {

    companion object {
        var instance: VpnService? = null

        const val PRIVATE_VLAN4_CLIENT = "172.19.0.1"
        const val PRIVATE_VLAN4_ROUTER = "172.19.0.2"
        const val FAKEDNS_VLAN4_CLIENT = "198.18.0.0"
        const val PRIVATE_VLAN6_CLIENT = "fdfe:dcba:9876::1"
        const val PRIVATE_VLAN6_ROUTER = "fdfe:dcba:9876::2"
        const val FAKEDNS_VLAN6_CLIENT = "fc00::"

        private fun <T> FileDescriptor.use(block: (FileDescriptor) -> T) = try {
            block(this)
        } finally {
            try {
                Os.close(this)
            } catch (_: ErrnoException) {
            }
        }
    }

    lateinit var conn: ParcelFileDescriptor
    private lateinit var tun: Tun2ray
    fun getTun(): Tun2ray? {
        if (!::tun.isInitialized) return null
        return tun
    }

    private var active = false
    private var metered = false

    @Volatile
    override var underlyingNetwork: Network? = null
        @RequiresApi(Build.VERSION_CODES.LOLLIPOP_MR1) set(value) {
            field = value
            if (active && Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP_MR1) {
                setUnderlyingNetworks(underlyingNetworks)
            }
        }
    private val underlyingNetworks
        get() = // clearing underlyingNetworks makes Android 9 consider the network to be metered
            if (Build.VERSION.SDK_INT == 28 && metered) null else underlyingNetwork?.let {
                arrayOf(it)
            }
    override var upstreamInterfaceName: String? = null
    private var worker: ProtectWorker? = null

    override suspend fun startProcesses() {
        worker = ProtectWorker().apply { start() }
        super.startProcesses()
        startVpn()
    }

    override var wakeLock: PowerManager.WakeLock? = null

    @SuppressLint("WakelockTimeout")
    override fun acquireWakeLock() {
        wakeLock = SagerNet.power.newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, "sagernet:vpn")
            .apply { acquire() }
    }

    @Suppress("EXPERIMENTAL_API_USAGE")
    override fun killProcesses() {
        runOnDefaultDispatcher {
            worker?.shutdown(this)
            worker = null
        }
        getTun()?.close()
        if (::conn.isInitialized) conn.close()
        super.killProcesses()
        persistAppStats()
        active = false
    }

    override fun onBind(intent: Intent) = when (intent.action) {
        SERVICE_INTERFACE -> super<BaseVpnService>.onBind(intent)
        else -> super<BaseService.Interface>.onBind(intent)
    }

    override val data = BaseService.Data(this)
    override val tag = "SagerNetVpnService"
    override fun createNotification(profileName: String) =
        ServiceNotification(this, profileName, "service-vpn")

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (DataStore.serviceMode == Key.MODE_VPN) {
            if (prepare(this) != null) {
                startActivity(
                    Intent(
                        this, VpnRequestActivity::class.java
                    ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                )
            } else return super<BaseService.Interface>.onStartCommand(intent, flags, startId)
        }
        stopRunner()
        return Service.START_NOT_STICKY
    }

    inner class NullConnectionException : NullPointerException(),
        BaseService.ExpectedException {
        override fun getLocalizedMessage() = getString(R.string.reboot_required)
    }

    private fun startVpn() {
        instance = this

        val profile = data.proxy!!.profile
        val builder = Builder().setConfigureIntent(SagerNet.configureIntent(this))
            .setSession(profile.displayName())
            .setMtu(DataStore.mtu)
        val useFakeDns = DataStore.enableFakeDns
        val ipv6Mode = DataStore.ipv6Mode

        builder.addAddress(PRIVATE_VLAN4_CLIENT, 30)
        if (useFakeDns) {
            builder.addAddress(FAKEDNS_VLAN4_CLIENT, 15)
        }

        if (ipv6Mode != IPv6Mode.DISABLE) {
            builder.addAddress(PRIVATE_VLAN6_CLIENT, 126)

            if (useFakeDns) {
                builder.addAddress(FAKEDNS_VLAN6_CLIENT, 18)
            }
        }

        if (DataStore.bypassLan && !DataStore.bypassLanInCoreOnly) {
            resources.getStringArray(R.array.bypass_private_route).forEach {
                val subnet = Subnet.fromString(it)!!
                builder.addRoute(subnet.address.hostAddress!!, subnet.prefixSize)
            }
            builder.addRoute(PRIVATE_VLAN4_ROUTER, 32)
            // https://issuetracker.google.com/issues/149636790
            if (ipv6Mode != IPv6Mode.DISABLE) {
                builder.addRoute("2000::", 3)
            }
        } else {
            builder.addRoute("0.0.0.0", 0)
            if (ipv6Mode != IPv6Mode.DISABLE) {
                builder.addRoute("::", 0)
            }
        }

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP_MR1) {
            builder.setUnderlyingNetworks(underlyingNetworks)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) builder.setMetered(metered)

        val packageName = packageName
        var proxyApps = DataStore.proxyApps
        var bypass = DataStore.bypass
        var workaroundSYSTEM = DataStore.tunImplementation == TunImplementation.SYSTEM
        var needBypassRootUid = workaroundSYSTEM || data.proxy!!.config.outboundTagsAll.values.any {
            it.nekoBean?.needBypassRootUid() == true || it.hysteriaBean?.protocol == HysteriaBean.PROTOCOL_FAKETCP
        }

        if (proxyApps || needBypassRootUid) {
            val individual = mutableSetOf<String>()
            val allApps by lazy {
                packageManager.getInstalledPackages(PackageManager.GET_PERMISSIONS).filter {
                    when (it.packageName) {
                        packageName -> false
                        "android" -> true
                        else -> it.requestedPermissions?.contains(Manifest.permission.INTERNET) == true
                    }
                }.map {
                    it.packageName
                }
            }
            if (proxyApps) {
                individual.addAll(DataStore.individual.split('\n').filter { it.isNotBlank() })
                if (bypass && needBypassRootUid) {
                    val individualNew = allApps.toMutableList()
                    individualNew.removeAll(individual)
                    individual.clear()
                    individual.addAll(individualNew)
                    bypass = false
                }
            } else {
                individual.addAll(allApps)
                bypass = false
            }

            val added = mutableListOf<String>()

            individual.apply {
                // Allow Matsuri itself using VPN.
                remove(packageName)
                if (!bypass) add(packageName)
            }.forEach {
                try {
                    if (bypass) {
                        builder.addDisallowedApplication(it)
                    } else {
                        builder.addAllowedApplication(it)
                    }
                    added.add(it)
                } catch (ex: PackageManager.NameNotFoundException) {
                    Logs.w(ex)
                }
            }

            if (bypass) {
                Logs.d("Add bypass: ${added.joinToString(", ")}")
            } else {
                Logs.d("Add allow: ${added.joinToString(", ")}")
            }
        }

        builder.addDnsServer(PRIVATE_VLAN4_ROUTER)

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q && DataStore.appendHttpProxy && DataStore.requireHttp) {
            builder.setHttpProxy(ProxyInfo.buildDirectProxy(LOCALHOST, DataStore.httpPort))
        }

        metered = DataStore.meteredNetwork
        active = true   // possible race condition here?
        if (Build.VERSION.SDK_INT >= 29) builder.setMetered(metered)
        conn = builder.establish() ?: throw NullConnectionException()

        val config = TunConfig().apply {
            fileDescriptor = conn.fd
            mtu = DataStore.mtu
            v2Ray = data.proxy!!.v2rayPoint
            iPv6Mode = ipv6Mode
            implementation = DataStore.tunImplementation
            sniffing = DataStore.trafficSniffing
            fakeDNS = DataStore.enableFakeDns
            debug = DataStore.enableLog
            dumpUID = data.proxy!!.config.dumpUid
            trafficStats = DataStore.appTrafficStatistics
            pCap = DataStore.enablePcap
            errorHandler = ErrorHandler {
                stopRunner(false, it)
            }
            localResolver = this@VpnService
            fdProtector = this@VpnService
        }

        tun = Libcore.newTun2ray(config)
    }

    // this is sekaiResolver
    override fun lookupIP(network: String, domain: String): String {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            return runBlocking {
                suspendCoroutine { continuation ->
                    val signal = CancellationSignal()
                    val callback = object : DnsResolver.Callback<Collection<InetAddress>> {
                        @Suppress("ThrowableNotThrown")
                        override fun onAnswer(answer: Collection<InetAddress>, rcode: Int) {
                            // libcore/v2ray.go
                            when {
                                answer.isNotEmpty() -> {
                                    continuation.tryResume((answer as Collection<InetAddress?>).mapNotNull { it?.hostAddress }
                                        .joinToString(","))
                                }
                                rcode == 0 -> {
                                    // fuck AAAA no record
                                    // features/dns/client.go
                                    continuation.tryResume("")
                                }
                                else -> {
                                    // Need return rcode
                                    // proxy/dns/dns.go
                                    continuation.tryResumeWithException(Exception("$rcode"))
                                }
                            }
                        }

                        override fun onError(error: DnsResolver.DnsException) {
                            continuation.tryResumeWithException(error)
                        }
                    }
                    val type = when {
                        network.endsWith("4") -> DnsResolver.TYPE_A
                        network.endsWith("6") -> DnsResolver.TYPE_AAAA
                        else -> null
                    }
                    if (type != null) {
                        DnsResolver.getInstance().query(
                            underlyingNetwork,
                            domain,
                            type,
                            DnsResolver.FLAG_EMPTY,
                            Dispatchers.IO.asExecutor(),
                            signal,
                            callback
                        )
                    } else {
                        DnsResolver.getInstance().query(
                            underlyingNetwork,
                            domain,
                            DnsResolver.FLAG_EMPTY,
                            Dispatchers.IO.asExecutor(),
                            signal,
                            callback
                        )
                    }
                }
            }
        } else {
            throw Exception("114514")
        }
    }

    val appStats = mutableListOf<AppStats>()

    override fun updateStats(stats: AppStats) {
        appStats.add(stats)
    }

    fun persistAppStats() {
        if (!DataStore.appTrafficStatistics) return
        val tun = getTun() ?: return
        appStats.clear()
        tun.readAppTraffics(this)
        val toUpdate = mutableListOf<StatsEntity>()
        val all = SagerDatabase.statsDao.all().associateBy { it.packageName }
        for (stats in appStats) {
            if (stats.nekoConnectionsJSON.isNotBlank()) continue
            val packageName = if (stats.uid >= 10000) {
                PackageCache.uidMap[stats.uid]?.iterator()?.next() ?: "android"
            } else {
                "android"
            }
            if (!all.containsKey(packageName)) {
                SagerDatabase.statsDao.create(
                    StatsEntity(
                        packageName = packageName,
                        tcpConnections = stats.tcpConnTotal,
                        udpConnections = stats.udpConnTotal,
                        uplink = stats.uplinkTotal,
                        downlink = stats.downlinkTotal
                    )
                )
            } else {
                val entity = all[packageName]!!
                entity.tcpConnections += stats.tcpConnTotal
                entity.udpConnections += stats.udpConnTotal
                entity.uplink += stats.uplinkTotal
                entity.downlink += stats.downlinkTotal
                toUpdate.add(entity)
            }
            if (toUpdate.isNotEmpty()) {
                SagerDatabase.statsDao.update(toUpdate)
            }

        }
    }

    override fun onRevoke() = stopRunner()

    override fun onDestroy() {
        super.onDestroy()
        data.binder.close()
    }

    private inner class ProtectWorker : ConcurrentLocalSocketListener(
        "ShadowsocksVpnThread",
        File(SagerNet.application.noBackupFilesDir, "protect_path")
    ) {
        override fun acceptInternal(socket: LocalSocket) {
            if (socket.inputStream.read() == -1) return
            val success = socket.ancillaryFileDescriptors?.single()?.use { fd ->
                underlyingNetwork.let { network ->
                    if (network != null) try {
                        network.bindSocket(fd)
                        return@let true
                    } catch (e: IOException) {
                        Logs.w(e)
                        return@let false
                    }
                    protect(fd.int)
                }
            } ?: false
            try {
                socket.outputStream.write(if (success) 0 else 1)
            } catch (_: IOException) {
            }        // ignore connection early close
        }
    }

}