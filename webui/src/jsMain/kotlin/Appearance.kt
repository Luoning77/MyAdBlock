// SPDX-License-Identifier: GPL-3.0-or-later
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

/** Browser-local appearance only; never changes DNS or device authorization. */
internal object UiAppearance {
 private val storage: dynamic = runCatching { js("window.localStorage") }.getOrNull()
 var mode by mutableStateOf(runCatching { storage.getItem("myadblock.appearance") as? String }.getOrNull()?.takeIf { it in listOf("system", "light", "dark") } ?: "system")
  private set
 fun select(value: String) {
  if (value !in listOf("system", "light", "dark")) return
  mode = value
  runCatching { storage.setItem("myadblock.appearance", value) }
 }
 fun label(): String = when (mode) { "light" -> "浅色"; "dark" -> "深色"; else -> "跟随系统" }
}
