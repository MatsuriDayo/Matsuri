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

package io.nekohasekai.sagernet.ui

import android.os.Bundle
import android.view.View
import android.view.ViewGroup
import androidx.core.view.isVisible
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.RecyclerView
import io.nekohasekai.sagernet.BuildConfig
import io.nekohasekai.sagernet.Key
import io.nekohasekai.sagernet.R
import io.nekohasekai.sagernet.aidl.AppStats
import io.nekohasekai.sagernet.database.DataStore
import io.nekohasekai.sagernet.databinding.LayoutTrafficConnBinding
import io.nekohasekai.sagernet.databinding.LayoutTrafficListBinding
import io.nekohasekai.sagernet.ktx.*
import io.nekohasekai.sagernet.utils.PackageCache
import moe.matsuri.nya.utils.Util
import org.json.JSONArray
import org.json.JSONObject

class ActiveConnFragment : Fragment(R.layout.layout_traffic_list) {

    lateinit var binding: LayoutTrafficListBinding
    lateinit var adapter: ActiveAdapter

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        binding = LayoutTrafficListBinding.bind(view)
        adapter = ActiveAdapter()
        binding.trafficList.layoutManager = FixedLinearLayoutManager(binding.trafficList)
        binding.trafficList.adapter = adapter
        emitStats(emptyList())
        (parentFragment as TrafficFragment).listeners.add(::emitStats)
    }


    fun emitStats(statsList: List<AppStats>) {
        if (statsList.isEmpty()) {
            runOnMainDispatcher {
                binding.holder.isVisible = true
                binding.trafficList.isVisible = false

                if (!DataStore.serviceState.started || DataStore.serviceMode != Key.MODE_VPN) {
                    binding.holder.text = getString(R.string.traffic_holder)
                } else if ((activity as MainActivity).connection.service?.trafficStatsEnabled != true) {
                    binding.holder.text = getString(R.string.app_statistics_disabled)
                } else {
                    binding.holder.text = getString(R.string.no_statistics)
                }
            }
            binding.trafficList.post {
                adapter.data = emptyList()
                adapter.notifyDataSetChanged()
            }
        } else {
            runOnMainDispatcher {
                binding.holder.isVisible = false
                binding.trafficList.isVisible = true
            }

            // 找到最后一个是 native 传过来的 json
            val json = JSONArray(statsList[statsList.size - 1].nekoConnectionsJSON)

            binding.trafficList.post {
                adapter.data = json.filterIsInstance()
                adapter.notifyDataSetChanged()
            }
        }
    }

    inner class ActiveAdapter : RecyclerView.Adapter<ActiveViewHolder>() {

        init {
            setHasStableIds(true)
        }

        lateinit var data: List<JSONObject>

        override fun getItemId(position: Int): Long {
            return data[position].optLong("ID")
        }

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ActiveViewHolder {
            return ActiveViewHolder(
                LayoutTrafficConnBinding.inflate(layoutInflater, parent, false)
            )
        }

        override fun onBindViewHolder(holder: ActiveViewHolder, position: Int) {
            holder.bind(data[position])
        }

        override fun getItemCount(): Int {
            if (!::data.isInitialized) return 0
            return data.size
        }
    }

    inner class ActiveViewHolder(val binding: LayoutTrafficConnBinding) : RecyclerView.ViewHolder(
        binding.root
    ) {

        fun bind(stats: JSONObject) {
            val tag = stats.optString("Tag")
            val start = stats.optLong("Start") * 1000
            val end = stats.optLong("End") * 1000
            if (tag == "freedom") return

            PackageCache.awaitLoadSync()
            val uid = stats.optInt("Uid")
            val packageName = if (uid > 1000) {
                PackageCache.uidMap[uid]?.iterator()?.next() ?: "android"
            } else if (uid == 0) {
                // including v2ray socks inbound
                BuildConfig.APPLICATION_ID
            } else {
                // including root
                "android"
            }
            val info = PackageCache.installedApps[packageName]
            if (info != null) runOnDefaultDispatcher {
                try {
                    val icon = info.loadIcon(app.packageManager)
                    onMainDispatcher {
                        binding.icon.setImageDrawable(icon)
                    }
                } catch (ignored: Exception) {
                }
            }

            var text1 = "\nStart: " + Util.timeStamp2Text(
                start
            )
            if (end > 0) text1 += "\nEnd: " + Util.timeStamp2Text(end)
            text1 += "\n" + tag

            var text2 = stats.optString("Dest")
            if (uid != 0) text2 += "\n" + PackageCache.loadLabel(packageName) + " ($uid)"

            binding.desc.text = text1
            binding.label.text = text2
            if (end <= 0) {
                binding.bottom.isVisible = true
                binding.isActive.text = "Active"
            } else {
                binding.bottom.isVisible = false
            }
        }

    }

}