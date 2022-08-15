package moe.matsuri.nya.neko

import android.content.pm.PackageInfo
import android.widget.Toast
import io.nekohasekai.sagernet.SagerNet
import io.nekohasekai.sagernet.database.DataStore
import io.nekohasekai.sagernet.plugin.PluginManager.loadString
import io.nekohasekai.sagernet.utils.PackageCache

object Plugins {
    const val AUTHORITIES_PREFIX_SEKAI_EXE = "io.nekohasekai.sagernet.plugin."
    const val AUTHORITIES_PREFIX_NEKO_EXE = "moe.matsuri.exe."
    const val AUTHORITIES_PREFIX_NEKO_PLUGIN = "moe.matsuri.plugin."

    const val METADATA_KEY_ID = "io.nekohasekai.sagernet.plugin.id"
    const val METADATA_KEY_EXECUTABLE_PATH = "io.nekohasekai.sagernet.plugin.executable_path"

    fun isExeOrPlugin(pkg: PackageInfo): Boolean {
        if (pkg.providers == null || pkg.providers.isEmpty()) return false
        val provider = pkg.providers[0] ?: return false
        val auth = provider.authority ?: return false
        return auth.startsWith(AUTHORITIES_PREFIX_SEKAI_EXE)
                || auth.startsWith(AUTHORITIES_PREFIX_NEKO_EXE)
                || auth.startsWith(AUTHORITIES_PREFIX_NEKO_PLUGIN)
    }

    fun preferExePrefix(): String {
        var prefix = AUTHORITIES_PREFIX_NEKO_EXE
        if (DataStore.exePreferProvider == 1) prefix = AUTHORITIES_PREFIX_SEKAI_EXE
        return prefix
    }

    fun displayExeProvider(pkgName: String): String {
        return if (pkgName.startsWith(AUTHORITIES_PREFIX_SEKAI_EXE)) {
            "SagerNet"
        } else if (pkgName.startsWith(AUTHORITIES_PREFIX_NEKO_EXE)) {
            "Matsuri"
        } else {
            "Unknown"
        }
    }

    fun getPlugin(pluginId: String): PackageInfo? {
        var pkgs = PackageCache.installedPluginPackages
            .map { it.value }
            .filter { it.providers[0].loadString(METADATA_KEY_ID) == pluginId }
        if (pkgs.isEmpty()) return null

        if (pkgs.size > 1) {
            val prefer = pkgs.filter {
                it.providers[0].authority.startsWith(preferExePrefix())
            }
            if (prefer.size == 1) pkgs = prefer
        }
        if (pkgs.size > 1) {
            val message = "Conflicting plugins found from: ${pkgs.joinToString { it.packageName }}"
            Toast.makeText(SagerNet.application, message, Toast.LENGTH_LONG).show()
        }
        return pkgs.single()
    }

}
