package moe.matsuri

import android.app.Activity
import android.content.Context
import android.util.Log
import android.view.Gravity
import android.view.View
import android.view.ViewGroup
import android.widget.ImageButton
import android.widget.LinearLayout
import androidx.coordinatorlayout.widget.CoordinatorLayout
import com.google.android.gms.ads.MobileAds
import com.google.android.gms.ads.AdListener
import com.google.android.gms.ads.AdRequest
import com.google.android.gms.ads.AdSize
import com.google.android.gms.ads.AdView
import com.google.android.gms.ads.LoadAdError
import com.google.android.gms.ads.appopen.AppOpenAd
import io.nekohasekai.sagernet.R
import io.nekohasekai.sagernet.ktx.dp2pxf
import io.nekohasekai.sagernet.ktx.isOss
import io.nekohasekai.sagernet.ktx.isPlay

object MoeAd {
    private const val LOG_TAG = "MoeAd"

    private const val APP_OPEN_TEST = "ca-app-pub-3940256099942544/3419835294"
    private const val APP_OPEN_RELEASE = "ca-app-pub-3940256099942544/3419835294"
    private val AD_APPOPEN_UNIT_ID = if (isOss || isPlay) APP_OPEN_RELEASE else APP_OPEN_TEST

    private const val BANNER_TEST = "ca-app-pub-3940256099942544/6300978111"
    private const val BANNER_RELEASE = "ca-app-pub-2318700711963667/2682266712"
    private val AD_BANNER_UNIT_ID = if (isOss || isPlay) BANNER_RELEASE else BANNER_TEST

    fun initialize(ctx: Context) {
        MobileAds.initialize(ctx)
    }

    fun showBannerAd(parentLayout: ViewGroup) {
        val ctx = parentLayout.context
        val req = AdRequest.Builder().build()
        val adview = AdView(ctx).apply {
            setAdSize(AdSize.BANNER)
            adUnitId = AD_BANNER_UNIT_ID
        }
        val layout = LinearLayout(ctx)
        val close = ImageButton(ctx, null, R.style.Widget_AppCompat_Button_Borderless).apply {
            setOnClickListener {
                layout.visibility = View.GONE
            }
            setImageResource(R.drawable.ic_navigation_close)
            visibility = View.GONE
        }
        layout.apply {
            layoutParams = CoordinatorLayout.LayoutParams(
                CoordinatorLayout.LayoutParams.WRAP_CONTENT,
                CoordinatorLayout.LayoutParams.WRAP_CONTENT
            ).apply {
                anchorId = R.id.fab
                anchorGravity = Gravity.CENTER_HORIZONTAL or Gravity.TOP
                translationY = -dp2pxf(32)
            }
            addView(adview.apply {
                adListener = BannerAdListener(close)
                loadAd(req)
            })
            addView(close)
        }
        parentLayout.addView(layout)
    }

    internal class BannerAdListener(val close: ImageButton) : AdListener() {
        override fun onAdFailedToLoad(p0: LoadAdError) {
            super.onAdFailedToLoad(p0)
            Log.d(LOG_TAG, "onAdFailedToLoad")
        }

        override fun onAdLoaded() {
            super.onAdLoaded()
            close.visibility = View.VISIBLE
        }
    }

    fun showAppOpenAd(activity: Activity) {
        val req = AdRequest.Builder().build()
        val callback = AppOpenAdCallback(activity)
        AppOpenAd.load(activity, AD_APPOPEN_UNIT_ID, req, callback)
    }

    internal class AppOpenAdCallback(
        val activity: Activity
    ) : AppOpenAd.AppOpenAdLoadCallback() {
        override fun onAdFailedToLoad(p0: LoadAdError) {
            super.onAdFailedToLoad(p0)
            Log.d(LOG_TAG, "onAdFailedToLoad")
        }

        override fun onAdLoaded(ad: AppOpenAd) {
            super.onAdLoaded(ad)
//            Log.d(LOG_TAG, "onAdLoaded")
            ad.show(activity)
        }
    }
}