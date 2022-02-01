package moe.matsuri.nya.neko

import android.annotation.SuppressLint
import android.os.Bundle
import com.takisoft.preferencex.PreferenceFragmentCompat
import io.nekohasekai.sagernet.R
import io.nekohasekai.sagernet.database.DataStore
import io.nekohasekai.sagernet.ktx.runOnIoDispatcher
import io.nekohasekai.sagernet.ui.profile.ProfileSettingsActivity
import moe.matsuri.nya.ui.Dialogs
import org.json.JSONArray

class NekoSettingActivity : ProfileSettingsActivity<NekoBean>() {

    lateinit var jsi: NekoJSInterface
    lateinit var jsip: NekoJSInterface.NekoProtocol
    lateinit var plgId: String
    lateinit var protocolId: String

    override fun createEntity() = NekoBean()

    override fun NekoBean.init() {
        if (!this@NekoSettingActivity::plgId.isInitialized) this@NekoSettingActivity.plgId = plgId
        if (!this@NekoSettingActivity::protocolId.isInitialized) this@NekoSettingActivity.protocolId = protocolId

        DataStore.sharedStorage = sharedStorage.toString()
    }

    override fun NekoBean.serialize() {
        // NekoBean from input
        plgId = this@NekoSettingActivity.plgId
        protocolId = this@NekoSettingActivity.protocolId

        sharedStorage = NekoBean.tryParseJSON(DataStore.sharedStorage)
        onSharedStorageSet()
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        intent?.getStringExtra("plgId")?.apply { plgId = this }
        intent?.getStringExtra("protocolId")?.apply { protocolId = this }
        super.onCreate(savedInstanceState)
    }

    @SuppressLint("SetJavaScriptEnabled", "JavascriptInterface")
    override fun PreferenceFragmentCompat.createPreferences(
        savedInstanceState: Bundle?,
        rootKey: String?,
    ) {
        addPreferencesFromResource(R.xml.neko_preferences)

        // Create a jsi
        jsi = NekoJSInterface(plgId)
        runOnIoDispatcher {
            try {
                jsi.init()
                jsip = jsi.switchProtocol(protocolId)
                jsi.jsObject.preferenceScreen = preferenceScreen

                // Because of the Preference problem, first require the KV and then inflate the UI
                jsip.setSharedStorage(DataStore.sharedStorage)
                jsip.requireSetProfileCache()

                val config = jsip.requirePreferenceScreenConfig()
                val pref = JSONArray(config)
                NekoPreferenceInflater.inflate(pref, preferenceScreen)

                jsip.onPreferenceCreated()
            } catch (e: Exception) {
                Dialogs.logExceptionAndShow(this@NekoSettingActivity, e) { finish() }
            }
        }
    }

    override suspend fun saveAndExit() {
        DataStore.sharedStorage = jsip.sharedStorageFromProfileCache()
        super.saveAndExit() // serialize & finish
    }

    override fun onDestroy() {
        jsi.destroy()
        super.onDestroy()
    }

}