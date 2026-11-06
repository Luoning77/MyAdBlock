// SPDX-License-Identifier: GPL-3.0-or-later
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import top.yukonga.miuix.kmp.basic.*
import top.yukonga.miuix.kmp.preference.ArrowPreference
import top.yukonga.miuix.kmp.preference.RadioButtonPreference
import top.yukonga.miuix.kmp.preference.RadioButtonLocation
import top.yukonga.miuix.kmp.theme.MiuixTheme

@Composable internal fun Group(title: String, content: @Composable ColumnScope.() -> Unit) {
 Column(Modifier.fillMaxWidth().padding(top = 12.dp)) {
  if (title.isNotEmpty()) SmallTitle(title, textColor = MiuixTheme.colorScheme.onSurfaceVariantSummary, insideMargin = PaddingValues(horizontal = 16.dp, vertical = 8.dp))
  Card(Modifier.fillMaxWidth(), content = content)
 }
}

@Composable internal fun FormGroup(title: String, content: @Composable ColumnScope.() -> Unit) {
 Group(title) { Column(Modifier.fillMaxWidth().padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp), content = content) }
}

@Composable internal fun Footnote(text: String, color: Color = Color.Unspecified) {
 Text(text, Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp), style = MiuixTheme.textStyles.footnote1,
  color = if (color == Color.Unspecified) MiuixTheme.colorScheme.onSurfaceVariantSummary else color)
}

@Composable internal fun IconTile(icon: ImageVector, color: Color, large: Boolean = false) {
 Icon(icon, null, Modifier.size(if (large) 48.dp else 24.dp), tint = color)
}

@Composable internal fun SettingRow(title: String, summary: String = "", value: String = "", onClick: (() -> Unit)? = null) {
 val trailing: @Composable RowScope.() -> Unit = {
  if (value.isNotEmpty()) Text(value, Modifier.widthIn(max = 100.dp), fontSize = 14.sp, maxLines = 2,
   overflow = TextOverflow.Ellipsis, color = MiuixTheme.colorScheme.onSurfaceVariantSummary)
 }
 if (onClick != null) ArrowPreference(title = title, summary = summary.takeIf { it.isNotEmpty() }, endActions = trailing, onClick = onClick)
 else BasicComponent(title = title, summary = summary.takeIf { it.isNotEmpty() }, endActions = if (value.isEmpty()) null else trailing)
}

@Composable internal fun Line(name: String, value: String) {
 Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(4.dp)) {
  Text(name, fontSize = 17.sp, fontWeight = FontWeight.Medium)
  Text(value, style = MiuixTheme.textStyles.body2, color = MiuixTheme.colorScheme.onSurfaceVariantSummary)
 }
}

@Composable internal fun Input(value: String, label: String, change: (String) -> Unit, password: Boolean = false) {
 TextField(value = value, onValueChange = change, label = label, modifier = Modifier.fillMaxWidth(),
  visualTransformation = if (password) PasswordVisualTransformation() else VisualTransformation.None,
  keyboardOptions = KeyboardOptions(keyboardType = if (password) KeyboardType.Password else KeyboardType.Text))
}

@Composable internal fun ActionButton(text: String, enabled: Boolean = true, primary: Boolean = true, modifier: Modifier = Modifier, onClick: () -> Unit) {
 TextButton(text, onClick = onClick, enabled = enabled, modifier = modifier.fillMaxWidth(),
  colors = if (primary) ButtonDefaults.textButtonColorsPrimary() else ButtonDefaults.textButtonColors())
}

@Composable internal fun ChoiceRow(title: String, summary: String = "", selected: Boolean, enabled: Boolean, onClick: () -> Unit) {
 RadioButtonPreference(title = title, summary = summary.takeIf { it.isNotEmpty() }, selected = selected,
  enabled = enabled, onClick = onClick, radioButtonLocation = RadioButtonLocation.End)
}

@Composable internal fun EmptyState(icon: ImageVector, title: String, description: String) {
 Column(Modifier.fillMaxWidth().padding(horizontal = 24.dp, vertical = 54.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(14.dp)) {
  Icon(icon, null, Modifier.size(42.dp), tint = MiuixTheme.colorScheme.onSurfaceVariantSummary)
  Text(title, fontSize = 20.sp, fontWeight = FontWeight.Medium)
  Text(description, fontSize = 14.sp, color = MiuixTheme.colorScheme.onSurfaceVariantSummary, textAlign = androidx.compose.ui.text.style.TextAlign.Center)
 }
}

internal fun encode(s: String): String = js("encodeURIComponent")(s) as String
internal fun formattedTime(raw: String): String {
 if (raw.isBlank() || raw == "—") return "尚无记录"
 return try {
  val date: dynamic = js("new Date(raw)")
  if (js("isNaN(date.getTime())") as Boolean) raw
  else date.toLocaleString("zh-CN", kotlin.js.json("hour12" to false)).toString()
 } catch (_: Throwable) { raw }
}
