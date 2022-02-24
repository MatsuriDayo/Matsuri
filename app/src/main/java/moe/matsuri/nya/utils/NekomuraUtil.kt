package moe.matsuri.nya.utils

import android.content.Context
import android.graphics.drawable.Drawable
import android.util.Base64
import androidx.appcompat.content.res.AppCompatResources
import io.nekohasekai.sagernet.SagerNet
import io.nekohasekai.sagernet.database.DataStore
import io.nekohasekai.sagernet.ktx.Logs
import libcore.HTTPResponse
import libcore.Libcore
import org.json.JSONObject
import java.io.ByteArrayOutputStream
import java.io.File
import java.util.zip.Deflater
import java.util.zip.Inflater
import kotlin.math.roundToInt

fun SagerNet.cleanWebview() {
    var pathToClean = "app_webview"
    if (isBgProcess) pathToClean += "_$process"
    try {
        val dataDir = filesDir.parentFile!!
        File(dataDir, "$pathToClean/BrowserMetrics").recreate(true)
        File(dataDir, "$pathToClean/BrowserMetrics-spare.pma").recreate(false)
    } catch (e: Exception) {
        Logs.e(e)
    }
}

fun File.recreate(dir: Boolean) {
    if (parentFile?.isDirectory != true) return
    if (dir && !isFile) {
        if (exists()) deleteRecursively()
        createNewFile()
    } else if (!dir && !isDirectory) {
        if (exists()) delete()
        mkdir()
    }
}

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
            val url = String(b64Decode(getSubString(content, "#UrlStart#", "#UrlEnd#")))
            val title = String(b64Decode(getSubString(content, "#TitleStart#", "#TitleEnd#")))

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
        return text.substring(zLen, yLen)
    }

    // Base64 for all

    fun b64EncodeUrlSafe(s: String): String {
        return b64EncodeUrlSafe(s.toByteArray())
    }

    fun b64EncodeUrlSafe(b: ByteArray): String {
        return String(Base64.encode(b, Base64.NO_PADDING or Base64.NO_WRAP or Base64.URL_SAFE))
    }

    // v2rayN Style
    fun b64EncodeOneLine(b: ByteArray): String {
        return String(Base64.encode(b, Base64.NO_WRAP))
    }

    fun b64EncodeDefault(b: ByteArray): String {
        return String(Base64.encode(b, Base64.DEFAULT))
    }

    fun b64Decode(b: String): ByteArray {
        var ret: ByteArray? = null

        // padding 自动处理，不用理
        // URLSafe 需要替换这两个，不要用 URL_SAFE 否则处理非 Safe 的时候会乱码
        val str = b.replace("-", "+").replace("_", "/")

        val flags = listOf(
            Base64.DEFAULT, // 多行
            Base64.NO_WRAP, // 单行
        )

        for (flag in flags) {
            try {
                ret = Base64.decode(str, flag)
            } catch (e: Exception) {
            }
            if (ret != null) return ret
        }

        throw IllegalStateException("Cannot decode base64")
    }

    fun zlibCompress(input: ByteArray, level: Int): ByteArray {
        // Compress the bytes
        // 1 to 4 bytes/char for UTF-8
        val output = ByteArray(input.size * 4)
        val compressor = Deflater(level).apply {
            setInput(input)
            finish()
        }
        val compressedDataLength: Int = compressor.deflate(output)
        compressor.end()
        return output.copyOfRange(0, compressedDataLength)
    }

    fun zlibDecompress(input: ByteArray): ByteArray {
        val inflater = Inflater()
        val outputStream = ByteArrayOutputStream()

        return outputStream.use {
            val buffer = ByteArray(1024)

            inflater.setInput(input)

            var count = -1
            while (count != 0) {
                count = inflater.inflate(buffer)
                outputStream.write(buffer, 0, count)
            }

            inflater.end()
            outputStream.toByteArray()
        }
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