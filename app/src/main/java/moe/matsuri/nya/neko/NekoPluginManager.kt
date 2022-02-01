package moe.matsuri.nya.neko

import io.nekohasekai.sagernet.SagerNet
import io.nekohasekai.sagernet.database.DataStore
import io.nekohasekai.sagernet.ktx.forEach
import io.nekohasekai.sagernet.utils.PackageCache
import okhttp3.internal.closeQuietly
import org.json.JSONObject
import java.io.File
import java.util.zip.ZipFile

object NekoPluginManager {
    const val managerVersion = 1

    val managedPlugins get() = DataStore.nekoPlugins.split("\n").filter { it.isNotBlank() }

    // plgID to plgConfig object
    fun getManagedPlugins(): Map<String, JSONObject> {
        val ret = mutableMapOf<String, JSONObject>()
        managedPlugins.forEach {
            tryGetPlgConfig(it)?.apply {
                ret[it] = this
            }
        }
        return ret
    }

    class Protocol(
        val protocolId: String, val plgId: String, val protocolConfig: JSONObject
    )

    fun getProtocols(): List<Protocol> {
        val ret = mutableListOf<Protocol>()
        getManagedPlugins().forEach { (t, u) ->
            u.optJSONArray("protocols")?.forEach { _, any ->
                if (any is JSONObject) {
                    val name = any.optString("protocolId")
                    ret.add(Protocol(name, t, any))
                }
            }
        }
        return ret
    }

    fun findProtocol(protocolId: String): Protocol? {
        getManagedPlugins().forEach { (t, u) ->
            u.optJSONArray("protocols")?.forEach { _, any ->
                if (any is JSONObject) {
                    if (protocolId == any.optString("protocolId")) {
                        return Protocol(protocolId, t, any)
                    }
                }
            }
        }
        return null
    }

    fun removeManagedPlugin(plgId: String) {
        DataStore.configurationStore.remove(plgId)
        val dir = File(SagerNet.application.filesDir.absolutePath + "/plugins/" + plgId)
        if (dir.exists()) {
            dir.deleteRecursively()
        }
    }

    fun extractPlugin(plgId: String) {
        val apkPath = PackageCache.installedApps[plgId]!!.publicSourceDir
        val zipFile = ZipFile(File(apkPath))
        val unzipDir = File(SagerNet.application.filesDir.absolutePath + "/plugins/" + plgId)
        if (unzipDir.exists()) unzipDir.deleteRecursively()
        unzipDir.mkdirs()
        for (entry in zipFile.entries()) {
            if (entry.name.startsWith("assets/")) {
                val relativePath = entry.name.removePrefix("assets/")
                val outFile = File(unzipDir, relativePath)
                if (entry.isDirectory) {
                    outFile.mkdirs()
                    continue
                }
                val input = zipFile.getInputStream(entry)
                outFile.outputStream().use {
                    input.copyTo(it)
                }
            }
        }
        zipFile.closeQuietly()
    }

    suspend fun installPlugin(plgId: String) {
        extractPlugin(plgId)
        NekoJSInterface.Default.destroyJsi(plgId)
        NekoJSInterface.Default.requireJsi(plgId).init()
        NekoJSInterface.Default.destroyJsi(plgId)
    }

    // reinstall plugin when it's versionCode changed
    suspend fun updateManagedPlugins() {
        getManagedPlugins().forEach { (t, u) ->
            val appVer = PackageCache.installedPackages[t]!!.versionCode
            val managedVer = u.optInt(PLUGIN_APP_VERSION)
            if (appVer != managedVer) {
                installPlugin(t)
            }
        }
    }

    const val PLUGIN_APP_VERSION = "_appVersionCode"

    // Return null if not managed
    fun tryGetPlgConfig(plgId: String): JSONObject? {
        return try {
            JSONObject(DataStore.configurationStore.getString(plgId)!!)
        } catch (e: Exception) {
            null
        }
    }

    fun updatePlgConfig(plgId: String, plgConfig: JSONObject) {
        plgConfig.put(PLUGIN_APP_VERSION, PackageCache.installedPackages[plgId]!!.versionCode)
        DataStore.configurationStore.putString(plgId, plgConfig.toString())
    }

    fun htmlPath(plgId: String): String {
        val htmlFile = File(SagerNet.application.filesDir.absolutePath + "/plugins/" + plgId)
        return htmlFile.absolutePath
    }

}