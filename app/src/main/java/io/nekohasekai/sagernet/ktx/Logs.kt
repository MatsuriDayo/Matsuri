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

package io.nekohasekai.sagernet.ktx

import libcore.Libcore
import java.io.InputStream
import java.io.OutputStream

object Logs {

    private fun mkTag(): String {
        val stackTrace = Thread.currentThread().stackTrace
        return stackTrace[4].className.substringAfterLast(".")
    }

    // level int use logrus.go

    fun v(message: String) {
        Libcore.nekoLogWrite(6, mkTag(), message)
    }

    fun v(message: String, exception: Throwable) {
        Libcore.nekoLogWrite(6, mkTag(), message + "\n" + exception.stackTraceToString())
    }

    fun d(message: String) {
        Libcore.nekoLogWrite(5, mkTag(), message)
    }

    fun d(message: String, exception: Throwable) {
        Libcore.nekoLogWrite(5, mkTag(), message + "\n" + exception.stackTraceToString())
    }

    fun i(message: String) {
        Libcore.nekoLogWrite(4, mkTag(), message)
    }

    fun i(message: String, exception: Throwable) {
        Libcore.nekoLogWrite(4, mkTag(), message + "\n" + exception.stackTraceToString())
    }

    fun w(message: String) {
        Libcore.nekoLogWrite(3, mkTag(), message)
    }

    fun w(message: String, exception: Throwable) {
        Libcore.nekoLogWrite(3, mkTag(), message + "\n" + exception.stackTraceToString())
    }

    fun w(exception: Throwable) {
        Libcore.nekoLogWrite(3, mkTag(), exception.stackTraceToString())
    }

    fun e(message: String) {
        Libcore.nekoLogWrite(2, mkTag(), message)
    }

    fun e(message: String, exception: Throwable) {
        Libcore.nekoLogWrite(2, mkTag(), message + "\n" + exception.stackTraceToString())
    }

    fun e(exception: Throwable) {
        Libcore.nekoLogWrite(2, mkTag(), exception.stackTraceToString())
    }

}

fun InputStream.use(out: OutputStream) {
    use { input ->
        out.use { output ->
            input.copyTo(output)
        }
    }
}