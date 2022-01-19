package io.nekohasekai.sagernet.utils

import android.util.Base64
import io.nekohasekai.sagernet.database.DataStore
import libcore.HTTPResponse
import libcore.Libcore
import org.json.JSONObject

object NekomuraUtil {

    // 0=Failed 1=No AD 2=AD
    fun updateAdurl(): Int {
        val response: HTTPResponse

        try {
            val client = Libcore.newHttpClient().apply {
                modernTLS()
                trySocks5(DataStore.socksPort)
            }

            response = client.newRequest().apply {
                setURL("https://api.github.com/repos/MatsuriDayo/nya/contents/ad1.txt?ref=main")
            }.execute()
        } catch (e: Exception) {
            return 0
        }

        try {
            val json = JSONObject(response.contentString)
            val content = String(Base64.decode(json.getString("content"), Base64.DEFAULT));

            // v1: very simple URL with base64
            val adurl = String(
                Base64.decode(
                    getSubString(
                        content, "#NekoStart#", "#NekoEnd#"
                    ), Base64.NO_PADDING
                )
            )

            if (adurl.startsWith("https://")) {
                DataStore.adurl = adurl
                return 2
            }
            return 1
        } catch (e: Exception) {
            // no result
            return 1
        }
    }

    /**
     * 取两个文本之间的文本值
     *
     * @param text  源文本 比如：欲取全文本为 12345
     * @param left  文本前面
     * @param right 后面文本
     * @return 返回 String
     */
    fun getSubString(text: String, left: String?, right: String?): String {
        var result = ""
        var zLen: Int
        if (left == null || left.isEmpty()) {
            zLen = 0
        } else {
            zLen = text.indexOf(left)
            if (zLen > -1) {
                zLen += left.length
            } else {
                zLen = 0
            }
        }
        var yLen = if (right == null) -1 else text.indexOf(right, zLen)
        if (yLen < 0 || right == null || right.isEmpty()) {
            yLen = text.length
        }
        result = text.substring(zLen, yLen)
        return result
    }

}