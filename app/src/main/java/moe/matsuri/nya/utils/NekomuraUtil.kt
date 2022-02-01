package moe.matsuri.nya.utils

import android.content.Context
import android.graphics.drawable.Drawable
import android.util.Base64
import androidx.appcompat.content.res.AppCompatResources
import io.nekohasekai.sagernet.SagerNet
import io.nekohasekai.sagernet.database.DataStore
import libcore.HTTPResponse
import libcore.Libcore
import org.json.JSONObject
import kotlin.math.roundToInt

object NekomuraUtil {

    class AdObject(
        var code: Int = 0, var url: String = "", var title: String = ""
    )

    // 0=Failed 1=No AD 2=AD
    fun updateAd(): AdObject {
        val ret = AdObject()
        val response: HTTPResponse

        try {
            val client = Libcore.newHttpClient().apply {
                modernTLS()
                trySocks5(DataStore.socksPort)
            }
            response = client.newRequest().apply {
                setURL("https://api.github.com/repos/MatsuriDayo/nya/contents/ad2.txt?ref=main")
            }.execute()
        } catch (e: Exception) {
            ret.code = 0
            return ret
        }

        try {
            val json = JSONObject(response.contentString)
            val content = String(Base64.decode(json.getString("content"), Base64.DEFAULT));

            // v2: very simple URL & title with base64
            val url = String(
                Base64.decode(
                    getSubString(
                        content, "#UrlStart#", "#UrlEnd#"
                    ), Base64.NO_PADDING
                )
            )
            val title = String(
                Base64.decode(
                    getSubString(
                        content, "#TitleStart#", "#TitleEnd#"
                    ), Base64.NO_PADDING
                )
            )

            if (url.startsWith("https://")) {
                ret.url = url
                ret.title = title
                ret.code = 2
                return ret
            }
            ret.code = 1
            return ret
        } catch (e: Exception) {
            // no result
            ret.code = 1
            return ret
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
        var result: String
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

    // Base64 for JS

    fun b64Encode(b: ByteArray): String {
        return String(Base64.encode(b, Base64.NO_PADDING or Base64.NO_WRAP or Base64.URL_SAFE))
    }

    fun b64Decode(b: String): ByteArray {
        return Base64.decode(b, Base64.NO_PADDING or Base64.NO_WRAP or Base64.URL_SAFE)
    }

}

// Context utils

fun Context.getDrawable(name: String?): Drawable? {
    val resourceId: Int = resources.getIdentifier(name, "drawable", packageName)
    return AppCompatResources.getDrawable(this, resourceId)
}

fun Context.dp2Pixel(sizeDp: Int): Int {
    val factor = resources.displayMetrics.density
    return (sizeDp * factor).roundToInt()
}