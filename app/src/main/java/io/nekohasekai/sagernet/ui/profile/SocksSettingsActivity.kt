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

package io.nekohasekai.sagernet.ui.profile

import android.os.Bundle
import androidx.preference.EditTextPreference
import androidx.preference.PreferenceCategory
import com.takisoft.preferencex.PreferenceFragmentCompat
import com.takisoft.preferencex.SimpleMenuPreference
import io.nekohasekai.sagernet.Key
import io.nekohasekai.sagernet.R
import io.nekohasekai.sagernet.database.DataStore
import io.nekohasekai.sagernet.database.preference.EditTextPreferenceModifiers
import io.nekohasekai.sagernet.fmt.socks.SOCKSBean

class SocksSettingsActivity : ProfileSettingsActivity<SOCKSBean>() {

    lateinit var securityCategory: PreferenceCategory

    override fun createEntity() = SOCKSBean()

    override fun SOCKSBean.init() {
        DataStore.profileName = name
        DataStore.serverAddress = serverAddress
        DataStore.serverPort = serverPort

        DataStore.serverProtocolVersion = protocol
        DataStore.serverUsername = username
        DataStore.serverPassword = password

        DataStore.serverSecurity = security
        DataStore.serverSNI = sni
        DataStore.serverALPN = alpn
        DataStore.serverCertificates = certificates
        DataStore.serverPinnedCertificateChain = pinnedPeerCertificateChainSha256
        DataStore.serverAllowInsecure = allowInsecure
    }

    override fun SOCKSBean.serialize() {
        name = DataStore.profileName
        serverAddress = DataStore.serverAddress
        serverPort = DataStore.serverPort

        protocol = DataStore.serverProtocolVersion
        username = DataStore.serverUsername
        password = DataStore.serverPassword

        security = DataStore.serverSecurity
        sni = DataStore.serverSNI
        alpn = DataStore.serverALPN
        certificates = DataStore.serverCertificates
        pinnedPeerCertificateChainSha256 = DataStore.serverPinnedCertificateChain
        allowInsecure = DataStore.serverAllowInsecure
    }

    override fun PreferenceFragmentCompat.createPreferences(
        savedInstanceState: Bundle?,
        rootKey: String?,
    ) {
        addPreferencesFromResource(R.xml.socks_preferences)
        findPreference<EditTextPreference>(Key.SERVER_PORT)!!.apply {
            setOnBindEditTextListener(EditTextPreferenceModifiers.Port)
        }
        val password = findPreference<EditTextPreference>(Key.SERVER_PASSWORD)!!.apply {
            summaryProvider = PasswordSummaryProvider
        }
        val protocol = findPreference<SimpleMenuPreference>(Key.SERVER_PROTOCOL)!!

        fun updateProtocol(version: Int) {
            password.isVisible = version == SOCKSBean.PROTOCOL_SOCKS5
        }

        updateProtocol(DataStore.serverProtocolVersion)
        protocol.setOnPreferenceChangeListener { _, newValue ->
            updateProtocol((newValue as String).toInt())
            true
        }

        // tls settings
        securityCategory = findPreference(Key.SERVER_SECURITY_CATEGORY)!!
        findPreference<SimpleMenuPreference>(Key.SERVER_SECURITY)!!.setOnPreferenceChangeListener { _, newValue ->
            updateTle(newValue as String)
            true
        }
        updateTle(DataStore.serverSecurity)
    }

    fun updateTle(tle: String) {
        val isTLS = tle == "tls"
        securityCategory.isVisible = isTLS
    }
}