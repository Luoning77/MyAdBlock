// SPDX-License-Identifier: GPL-3.0-or-later
@file:OptIn(androidx.compose.ui.ExperimentalComposeUiApi::class, org.jetbrains.compose.resources.ExperimentalResourceApi::class)
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.tween
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalWindowInfo
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.ComposeViewport
import kotlinx.coroutines.*
import kotlin.js.Promise
import kotlin.js.json
import top.yukonga.miuix.kmp.basic.*
import top.yukonga.miuix.kmp.overlay.OverlayBottomSheet
import top.yukonga.miuix.kmp.theme.*
import top.yukonga.miuix.kmp.icon.MiuixIcons
import top.yukonga.miuix.kmp.icon.extended.*
import androidx.compose.ui.text.font.FontFamily
import org.jetbrains.compose.resources.preloadFont
import org.jetbrains.compose.resources.ExperimentalResourceApi
import myadblockwebui.generated.resources.Res
import myadblockwebui.generated.resources.misans_vf

private val browser: dynamic = js("window")
private val doc: dynamic = js("document")
private external fun fetch(url: String, options: dynamic): Promise<dynamic>
internal fun str(o: dynamic, key: String, default: String = "—"): String = if (o == null || o[key] == null) default else o[key].toString()
internal fun bool(o: dynamic, key: String): Boolean = o != null && o[key] == true
internal fun array(o: dynamic): List<dynamic> = if (o == null) emptyList() else (o as Array<dynamic>).toList()
internal object API {
 var csrf = ""
 var authorized by mutableStateOf(false)
 var logged by mutableStateOf(false)
 var grant: String = ""
 suspend fun call(path: String, body: dynamic = null): dynamic {
  val opts = json("credentials" to "same-origin", "method" to if (body == null) "GET" else "POST")
  if (body != null) { opts["body"] = JSON.stringify(body); opts["headers"] = json("Content-Type" to "application/json", "X-CSRF-Token" to csrf) }
  val response = fetch("/api/$path", opts).await()
  val raw = (response.text() as Promise<String>).await()
  if (response.status == 401) { logged = false; authorized = false; throw Exception("会话已过期或核心未就绪，请重新登录") }
  if (response.status == 403) { authorized = false; throw Exception("操作未授权。请在 Root 管理器重新点击“操作”后再试。") }
  if (response.ok != true) { val message = try { str(JSON.parse<dynamic>(raw), "error", "请求失败") } catch (_: Throwable) { "服务请求失败（${response.status}）" }; throw Exception(message) }
  return if (raw.isEmpty()) json() else JSON.parse<dynamic>(raw)
 }
 suspend fun session() { val r = call("session"); csrf = str(r, "csrf", ""); logged = true; authorized = bool(r, "authorized"); if (grant.isNotEmpty()) { val token = grant; grant = ""; call("authorize", json("token" to token)); authorized = true } }
}
fun main() {
 val hash = browser.location.hash.toString()
 if (hash.startsWith("#grant=")) API.grant = hash.removePrefix("#grant=")
 browser.history.replaceState(null, "", browser.location.pathname)
 browser.addEventListener("hashchange", { _: dynamic -> if(browser.location.hash.toString().startsWith("#grant=")) browser.location.reload() })

 ComposeViewport { App() }
}

internal typealias Action = (suspend () -> Unit) -> Unit

@Composable private fun App() {
 val scope = rememberCoroutineScope()
 var page by remember { mutableStateOf(0) }
 var detail by remember { mutableStateOf("") }
 val parentRoutes = remember { mutableStateListOf<Pair<Int, String>>() }
 var motionDirection by remember { mutableStateOf(1f) }
 val enterProgress = remember { Animatable(1f) }
 val reduceMotion = remember { browser.matchMedia("(prefers-reduced-motion: reduce)").matches == true }
 var state by remember { mutableStateOf<dynamic>(null) }
 var stats by remember { mutableStateOf<dynamic>(null) }
 var stale by remember { mutableStateOf(true) }
 var error by remember { mutableStateOf("") }
 var connectionError by remember { mutableStateOf("") }
 var busy by remember { mutableStateOf(false) }
 var notice by remember { mutableStateOf("") }
 var pauseSheet by remember { mutableStateOf(false) }
 var pauseSeconds by remember { mutableStateOf(300) }
 var remaining by remember { mutableStateOf(0) }
 suspend fun refresh() {
  try {
   val nextState = API.call("status")
   val nextStats = API.call("stats?recent=" + (((js("new Date()").getHours() as Int) + 1) * 3600000))
   val session = API.call("session")
   state = nextState; stats = nextStats; stale = false; connectionError = ""
   remaining = str(state, "remaining", "0").toIntOrNull() ?: 0
   API.authorized = bool(session, "authorized")
  } catch (e: Throwable) { stale = true; throw e }
 }
 fun action(work: suspend () -> Unit) {
  if (busy) return
  scope.launch {
   busy = true; error = ""; notice = ""
   try { work(); if (API.logged) refresh(); notice = "操作已完成" }
   catch (e: Throwable) { error = e.message ?: "操作失败，请重新打开或使用应急入口" }
   finally { busy = false }
  }
 }
 fun navigate(next: Int) { motionDirection = if (next >= page) 1f else -1f; parentRoutes.clear(); page = next; detail = ""; notice = "" }
 fun restoreParent() { if (parentRoutes.isNotEmpty()) { val previous = parentRoutes.removeAt(parentRoutes.lastIndex); motionDirection = -1f; page = previous.first; detail = previous.second } else detail = "" }
 fun back() { if (parentRoutes.isNotEmpty()) browser.history.back() else detail = "" }
 fun openRoute(next: Int, section: String) {
  if (page == next && detail == section) return
  parentRoutes.add(page to detail); browser.history.pushState(json("myadblock" to true), "", browser.location.pathname)
  motionDirection = 1f; page = next; detail = section; notice = ""
 }
 fun openSettings(section: String) { openRoute(3, section) }
 fun openDetail(section: String) { if (section.isEmpty()) back() else openRoute(page, section) }
 DisposableEffect(Unit) {
  val listener: (dynamic) -> Unit = { if (API.logged) restoreParent() }
  browser.addEventListener("popstate", listener)
  onDispose { browser.removeEventListener("popstate", listener) }
 }
 LaunchedEffect(page, detail, API.logged) {
  if (reduceMotion) enterProgress.snapTo(1f)
  else { enterProgress.snapTo(0f); enterProgress.animateTo(1f, tween(240, easing = FastOutSlowInEasing)) }
 }
 LaunchedEffect(Unit) { try { API.session() } catch (_: Throwable) { API.logged = false } }
 LaunchedEffect(API.logged) {
  if (API.logged) while (isActive) {
   try { refresh() } catch (e: Throwable) { connectionError = e.message ?: "本地服务失联，状态未知。请用应急命令暂停接管。" }
   delay(5000)
  } else { page = 0; detail = ""; parentRoutes.clear(); state = null; stats = null; stale = true }
 }
 LaunchedEffect(remaining) { if (remaining > 0) { delay(1000); remaining-- } }
 LaunchedEffect(notice) { if (notice.isNotEmpty()) { delay(3500); notice = "" } }
 val labels = listOf("首页", "查询日志", "过滤规则", "设置")
 val icons = listOf(MiuixIcons.Home, MiuixIcons.ListView, MiuixIcons.Filter, MiuixIcons.Settings)
 val detailTitles = mapOf("appearance" to "外观设置", "session" to "管理会话", "dns" to "DNS 设置", "network" to "网络诊断", "security" to "账户与安全", "backup" to "备份与恢复", "about" to "关于 MyAdBlock", "user-rules" to "自定义规则", "add-filter" to "添加订阅")
 val title = if (!API.logged) "MyAdBlock" else detailTitles[detail] ?: if (page == 0) "MyAdBlock" else labels[page]
 val scroll = key(page, detail) { MiuixScrollBehavior() }
 val contentScroll = remember(page, detail) { ScrollState(0) }
 val font by preloadFont(Res.font.misans_vf)
 if (font == null) return
 SideEffect { doc.getElementById("loading")?.remove() }
 val family = FontFamily(font!!)
 val dark = when (UiAppearance.mode) { "light" -> false; "dark" -> true; else -> isSystemInDarkTheme() }
 val controller = remember(UiAppearance.mode, dark) { ThemeController(colorSchemeMode = when (UiAppearance.mode) { "light" -> ColorSchemeMode.Light; "dark" -> ColorSchemeMode.Dark; else -> ColorSchemeMode.System }, isDark = dark) }
 val base = defaultTextStyles()
 val typography = base.copy(main=base.main.copy(fontFamily=family),paragraph=base.paragraph.copy(fontFamily=family),body1=base.body1.copy(fontFamily=family),body2=base.body2.copy(fontFamily=family),button=base.button.copy(fontFamily=family),footnote1=base.footnote1.copy(fontFamily=family),footnote2=base.footnote2.copy(fontFamily=family),headline1=base.headline1.copy(fontFamily=family),headline2=base.headline2.copy(fontFamily=family),subtitle=base.subtitle.copy(fontFamily=family),title1=base.title1.copy(fontFamily=family),title2=base.title2.copy(fontFamily=family),title3=base.title3.copy(fontFamily=family),title4=base.title4.copy(fontFamily=family))
 MiuixTheme(controller = controller, textStyles = typography) {
  val colors = MiuixTheme.colorScheme
  MiuixTheme(colors = if (dark) colors else colors.copy(error = Color(0xFFC5392B))) {
  Scaffold(
   modifier = Modifier.nestedScroll(scroll.nestedScrollConnection),
   topBar = {
    Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
     TopAppBar(title = title, largeTitle = title, modifier = Modifier.widthIn(max = 720.dp).fillMaxWidth(), scrollBehavior = scroll,
      navigationIcon = { if (detail.isNotEmpty()) IconButton(onClick = { back() }, modifier = Modifier.padding(start = 12.dp)) { Icon(MiuixIcons.Back, "返回", Modifier.size(24.dp)) } },
      actions = { if (API.logged && !API.authorized) TextButton("只读", onClick = { openSettings("session") }, modifier = Modifier.padding(end = 12.dp), minWidth = 48.dp, minHeight = 28.dp, insideMargin = PaddingValues(horizontal = 10.dp, vertical = 5.dp), textStyle = MiuixTheme.textStyles.footnote1) })
    }
   },
   bottomBar = {
    if (API.logged && detail.isEmpty()) Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
     NavigationBar(modifier = Modifier.widthIn(max = 720.dp).fillMaxWidth(), showDivider = false) {
      labels.forEachIndexed { i, label -> NavigationBarItem(selected = page == i, onClick = { navigate(i) }, icon = icons[i], label = label) }
     }
    }
   }, containerColor = MiuixTheme.colorScheme.surface
  ) { padding ->
   Box(Modifier.fillMaxSize().padding(padding), contentAlignment = Alignment.TopCenter) {
    Column(Modifier.widthIn(max = 720.dp).fillMaxWidth().graphicsLayer { translationX = (1f - enterProgress.value) * 18.dp.toPx() * motionDirection; alpha = 0.85f + 0.15f * enterProgress.value }.verticalScroll(contentScroll).padding(horizontal = 12.dp).padding(top = 12.dp, bottom = 24.dp)) {
     val visibleError = listOf(error, connectionError).filter { it.isNotEmpty() }.distinct().joinToString("\n")
     if (visibleError.isNotEmpty()) {
      FormGroup("需要处理") { Text(visibleError, color = MiuixTheme.colorScheme.error); Text("网页不可用时，在 Root 终端执行 sh /data/adb/modules/myadblock/emergency.sh", fontSize = 13.sp) }
     }
     if (notice.isNotEmpty()) Footnote(notice, color = MiuixTheme.colorScheme.primary)
     if (!API.logged) Login(!busy, ::action)
     else {
      val enabled = API.authorized && !busy
      when (page) {
       0 -> Home(JSON.stringify(state), JSON.stringify(stats), stale, remaining, enabled,
        onPause = { pauseSeconds = 300; pauseSheet = true }, onResume = { action { API.call("resume", json()) } },
        onLogs = { navigate(1) }, onRules = { navigate(2) }, onNetwork = { openSettings("network") })
       1 -> Logs(enabled, ::action)
       2 -> Rules(enabled, if (stale) "" else str(state, "mode", ""), detail, ::openDetail, ::action)
       3 -> Settings(if (stale) "null" else JSON.stringify(state), enabled, busy, detail, ::openDetail, ::action)
      }
     }
    }
   }
   OverlayBottomSheet(show = pauseSheet, title = "暂停防护", onDismissRequest = { pauseSheet = false },
    startAction = { IconButton(onClick = { pauseSheet = false }) { Icon(MiuixIcons.Close, "关闭暂停选项", Modifier.size(22.dp)) } }) {
    val windowHeight = with(LocalDensity.current) { LocalWindowInfo.current.containerSize.height.toDp() }
    Column(Modifier.fillMaxWidth().heightIn(max = (windowHeight * 0.75f - 120.dp).coerceIn(220.dp, 560.dp))) {
     Column(Modifier.fillMaxWidth().weight(1f, fill = false).verticalScroll(rememberScrollState())) {
     Footnote("暂停过滤和 IPv4 / IPv6 接管。私人 DNS 保持关闭，重启手机会取消暂停。")
     if (!API.authorized) Footnote("当前为只读预览。请从 Root 管理器重新打开，获得授权后才可确认暂停。")
     val options = listOf("30 秒" to 30, "1 分钟" to 60, "5 分钟" to 300, "10 分钟" to 600, "1 小时" to 3600, "24 小时" to 86400, "无限期" to 0)
     options.forEach { (name, seconds) -> ChoiceRow(name, if (seconds == 300) "常用" else "", pauseSeconds == seconds, !busy && API.authorized) { pauseSeconds = seconds } }
     }
     ActionButton(if (API.authorized) "确认暂停" else "需要设备授权", API.authorized && !busy, modifier = Modifier.padding(top = 12.dp)) {
      pauseSheet = false; action { API.call("pause", json("seconds" to pauseSeconds)) }
     }
    }
   }
  }
  }
 }
}

@Composable private fun Login(enabled: Boolean, action: Action) {
 var password by remember { mutableStateOf("root") }
 Spacer(Modifier.height(20.dp))
 IconTile(MiuixIcons.Lock, MiuixTheme.colorScheme.primary, large = true)
 Text("管理本机防护", Modifier.padding(top = 20.dp), fontSize = 26.sp)
 Footnote("登录后查看查询记录、管理过滤规则与 DNS 设置。")
 FormGroup("本机账户") {
  Line("用户名", "root")
  Input(password, "密码", { password = it }, password = true)
  ActionButton("登录", enabled) { action { val r = API.call("login", json("username" to "root", "password" to password)); API.csrf = str(r, "csrf", ""); API.logged = true; API.session(); password = "" } }
 }
 Footnote("默认账户 root / root。默认密码公开，请登录后在“账户与安全”中修改；随机端口不能替代密码保护。")
}
