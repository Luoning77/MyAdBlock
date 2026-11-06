// SPDX-License-Identifier: GPL-3.0-or-later
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.*
import kotlin.js.json
import top.yukonga.miuix.kmp.basic.*
import top.yukonga.miuix.kmp.preference.SwitchPreference
import top.yukonga.miuix.kmp.theme.MiuixTheme
import top.yukonga.miuix.kmp.icon.MiuixIcons
import top.yukonga.miuix.kmp.icon.extended.*

@Composable internal fun Home(
 rawState: String, rawStats: String, stale: Boolean, remaining: Int, enabled: Boolean,
 onPause: () -> Unit, onResume: () -> Unit, onLogs: () -> Unit, onRules: () -> Unit, onNetwork: () -> Unit,
) {
 val state: dynamic = remember(rawState) { JSON.parse<dynamic>(rawState) }
 val stats: dynamic = remember(rawStats) { JSON.parse<dynamic>(rawStats) }
 val mode = str(state, "state", "UNKNOWN")
 val paused = mode == "USER_PAUSED"
 val uncertain = stale || mode !in listOf("PUBLIC_DOH", "SYSTEM_DNS_FALLBACK", "USER_PAUSED")
 val tint = when { uncertain -> MiuixTheme.colorScheme.error; paused -> MiuixTheme.colorScheme.onSurfaceVariantSummary; else -> MiuixTheme.colorScheme.primary }
 val title = when { stale -> "状态未知"; paused -> "防护已暂停"; mode == "PUBLIC_DOH" -> "公共 DoH 防护"; mode == "SYSTEM_DNS_FALLBACK" -> "系统 DNS 回退"; else -> "安全旁路" }
 val description = when {
  stale -> "暂时无法确认服务状态，请检查本地服务。"
  paused -> "过滤与 DNS 接管均已暂停。"
  mode == "PUBLIC_DOH" -> "正在通过公共加密 DNS 处理查询。"
  mode == "SYSTEM_DNS_FALLBACK" -> "公共上游暂不可用，已回退到系统 DNS。"
  else -> "当前未接管 DNS，请查看网络诊断。"
 }
 val statusBackground = when {
  uncertain -> MiuixTheme.colorScheme.errorContainer.copy(alpha = 0.45f)
  paused -> MiuixTheme.colorScheme.surfaceContainer
  else -> MiuixTheme.colorScheme.primaryContainer.copy(alpha = 0.35f)
 }
 Card(Modifier.fillMaxWidth(), colors = CardDefaults.defaultColors(color = statusBackground)) {
  Box(Modifier.fillMaxWidth()) {
   Box(Modifier.matchParentSize().offset(x = 28.dp, y = 22.dp), contentAlignment = Alignment.BottomEnd) {
    Icon(if (paused) MiuixIcons.Pause else if (uncertain) MiuixIcons.Info else MiuixIcons.Filter, null, Modifier.size(136.dp), tint = tint.copy(alpha = 0.10f))
   }
   Column(Modifier.fillMaxWidth().padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
    Text(title, fontSize = 20.sp, fontWeight = FontWeight.SemiBold)
    Text(description, style = MiuixTheme.textStyles.body2, color = MiuixTheme.colorScheme.onSurfaceVariantSummary)
    val reason = if (stale) "" else str(state, "error", "")
    if (reason.isNotEmpty()) Text(reason, fontSize = 13.sp, color = MiuixTheme.colorScheme.error)
    if (paused) Text(when { remaining < 0 -> "无限期暂停 · 等待手动恢复"; remaining == 0 -> "已到期，正在检查安全恢复条件"; else -> "${remaining / 60} 分 ${remaining % 60} 秒后尝试恢复" }, style = MiuixTheme.textStyles.footnote1, color = MiuixTheme.colorScheme.onSurfaceVariantSummary)
    TextButton(if (!API.authorized) "查看暂停选项" else if (paused) "恢复防护" else "暂停防护",
     enabled = enabled || !API.authorized, colors = if (API.authorized && paused) ButtonDefaults.textButtonColorsPrimary() else ButtonDefaults.textButtonColors(),
     onClick = if (!API.authorized) onPause else if (paused) onResume else onPause)
   }
  }
 }
 Group("今日查询") {
  val hours = (js("new Date()").getHours() as Int) + 1
  fun today(key: String): String {
   val values = array(if (stats == null) null else stats[key])
   return if (values.isEmpty() || stale) "—" else values.takeLast(hours).sumOf { (it as? Number)?.toLong() ?: 0L }.toString()
  }
  Row(Modifier.fillMaxWidth().clickable(onClick = onLogs).padding(16.dp), horizontalArrangement = Arrangement.spacedBy(24.dp)) {
   Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(8.dp)) {
    Text("DNS 查询", fontSize = 13.sp, color = MiuixTheme.colorScheme.onSurfaceVariantSummary)
    Text(today("dns_queries"), fontSize = 32.sp, fontWeight = FontWeight.Medium)
   }
   Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(8.dp)) {
    Text("已拦截", fontSize = 13.sp, color = MiuixTheme.colorScheme.onSurfaceVariantSummary)
    Text(today("blocked_filtering"), fontSize = 32.sp, fontWeight = FontWeight.Medium, color = MiuixTheme.colorScheme.primary)
   }
  }
 }
 Footnote(if (stale) "暂时没有可确认的统计数据。" else "仅统计到达本机核心的查询，按本地今日汇总。")
 Group("运行概况") {
  SettingRow("过滤核心", if (stale) "状态未知" else if (bool(state, "core")) "AdGuard Home 运行中" else "AdGuard Home 未就绪", if (stale) "未知" else if (bool(state, "protection")) "过滤开启" else "过滤关闭")
  SettingRow("DNS 接管", if (stale) "状态未知" else "IPv4 ${if (bool(state, "ipv4")) "已接管" else "未接管"} · IPv6 ${if (bool(state, "ipv6")) "已接管" else "放行"}", onClick = onNetwork)
  SettingRow("过滤预设", "管理订阅与自定义规则", if (stale) "未知" else when(str(state,"mode")) { "strict" -> "严格"; "daily" -> "日常"; else -> "未知" }, onClick = onRules)
 }
 Footnote("核心运行不等于所有流量已接管；IPv6 放行时覆盖不完整。")
}

@Composable internal fun Logs(enabled: Boolean, action: Action) {
 var search by remember { mutableStateOf("") }
 var appliedSearch by remember { mutableStateOf("") }
 var filter by remember { mutableStateOf("all") }
 var data by remember { mutableStateOf<dynamic>(null) }
 var loading by remember { mutableStateOf(true) }
 var message by remember { mutableStateOf("") }
 val scope = rememberCoroutineScope()
 var loadJob by remember { mutableStateOf<Job?>(null) }
 var requestId by remember { mutableStateOf(0) }
 fun load(older: String = "") {
  val id = ++requestId
  loadJob?.cancel()
  val query = search; val kind = filter
  loadJob = scope.launch {
   loading = true; message = ""
   try { data = API.call("querylog?search=${encode(query)}&filter=$kind&older=${encode(older)}"); appliedSearch = query }
   catch (e: CancellationException) { throw e }
   catch (e: Throwable) { message = e.message ?: "查询失败，请重试" }
   finally { if (requestId == id) loading = false }
  }
 }
 LaunchedEffect(Unit) { load() }
 FormGroup("") {
  Input(search, "搜索域名", { search = it })
  ActionButton("搜索查询记录", !loading, primary = false) { load() }
 }
 Row(Modifier.fillMaxWidth().padding(top = 16.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
  listOf("all" to "全部查询", "filtered" to "已拦截").forEach { (key, name) ->
   TextButton(name, onClick = { filter = key; load() }, modifier = Modifier.weight(1f),
    colors = if (filter == key) ButtonDefaults.textButtonColorsPrimary() else ButtonDefaults.textButtonColors())
  }
 }
 if (message.isNotEmpty()) Footnote(message, color = MiuixTheme.colorScheme.error)
 if (loading) Footnote("正在读取查询记录…")
 val entries = array(if (data == null) null else data.data)
 if (entries.isEmpty() && !loading && message.isEmpty()) EmptyState(MiuixIcons.Search,
  if (appliedSearch.isNotEmpty() || filter != "all") "没有匹配的查询" else "还没有查询记录",
  if (appliedSearch.isNotEmpty() || filter != "all") "试试其他域名，或切换到全部查询。" else "设备产生 DNS 查询后会显示在这里。默认仅在内存保留日志。")
 entries.forEach { item ->
  val domain = str(item.question, "name")
  val blocked = str(item, "reason", "").startsWith("Filtered")
  FormGroup("") {
   Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(5.dp)) {
     Text(domain, fontSize = 17.sp)
     Text(formattedTime(str(item,"time","")), fontSize = 12.sp, color = MiuixTheme.colorScheme.onSurfaceVariantSummary)
    }
    Text(if (blocked) "已拦截" else "未拦截", fontSize = 12.sp, color = if (blocked) MiuixTheme.colorScheme.error else MiuixTheme.colorScheme.onSurfaceVariantSummary)
   }
   Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
    TextButton("允许", modifier = Modifier.weight(1f), enabled = enabled, onClick = { action { API.call("domain", json("domain" to domain, "allow" to true)) } })
    TextButton("阻止", modifier = Modifier.weight(1f), enabled = enabled, onClick = { action { API.call("domain", json("domain" to domain, "allow" to false)) } })
   }
  }
 }
 if (entries.isNotEmpty()) {
  Footnote("每页最多 50 条查询，仅展示本机核心收到的记录。")
  ActionButton("下一页", !loading, primary = false) { load(str(entries.last(),"time","")) }
 }
}

@Composable internal fun Rules(enabled: Boolean, mode: String, detail: String, open: (String) -> Unit, action: Action) {
 var data by remember { mutableStateOf<dynamic>(null) }
 var rules by remember { mutableStateOf("") }
 var name by remember { mutableStateOf("") }
 var url by remember { mutableStateOf("") }
 var error by remember { mutableStateOf("") }
 var loading by remember { mutableStateOf(true) }
 suspend fun load() { data = API.call("filters"); rules = array(data.user_rules).joinToString("\n") { it.toString() }; loading = false }
 LaunchedEffect(Unit) { try { load() } catch(e: Throwable) { loading = false; error = e.message ?: "读取规则失败" } }
 if (error.isNotEmpty()) Footnote(error, color = MiuixTheme.colorScheme.error)
 when (detail) {
  "user-rules" -> {
   FormGroup("黑名单与白名单") {
    Text("每行一条 AdGuard Home DNS 规则。", fontSize = 14.sp)
    Input(rules, "用户规则", { rules = it })
    ActionButton("保存规则", enabled && !loading) { action { API.call("rules", json("rules" to rules.lines().filter { it.isNotBlank() }.toTypedArray())); load() } }
   }
   Footnote("白名单示例：@@||example.com^\n保存会替换当前自定义规则，请保留需要的内容。")
  }
  "add-filter" -> {
   FormGroup("新订阅") {
    Input(name, "订阅名称", { name = it }); Input(url, "HTTPS 订阅地址", { url = it })
    ActionButton("验证并添加", enabled && name.isNotBlank() && url.isNotBlank()) { action { API.call("filter/add", json("name" to name, "url" to url)); load(); name = ""; url = ""; open("") } }
   }
   Footnote("添加前会检查订阅内容。格式校验失败不会替换现有规则。")
  }
  else -> {
   Group("拦截预设") {
    ChoiceRow("日常", "均衡使用，适合日常网络", mode == "daily", enabled) { if (mode != "daily") action { API.call("mode",json("mode" to "daily")); load() } }
    ChoiceRow("严格", "使用严格规则与并行查询", mode == "strict", enabled) { if (mode != "strict") action { API.call("mode",json("mode" to "strict")); load() } }
   }
   Footnote("两份 AWAvenue 规则互斥启用，切换预设会保留自定义规则。")
   Group("过滤订阅") {
    val subscriptions = array(if (data == null) null else data.filters)
    if (loading)
    SettingRow("正在读取订阅…")
    else if (subscriptions.isEmpty())
    SettingRow("暂无订阅", "添加 HTTPS 订阅后会显示在这里")
    subscriptions.forEach { f ->
      SettingRow(str(f,"name"), "${str(f,"rules_count","0")} 条规则 · ${formattedTime(str(f,"last_updated","离线快照"))}", if (bool(f,"enabled")) "已开启" else "已关闭")
     val id = str(f,"id","0").toLongOrNull() ?: 0
     if (id > 3) TextButton("删除此订阅", enabled = enabled, modifier = Modifier.padding(start = 16.dp, bottom = 12.dp), onClick = { action { API.call("filter/remove",json("id" to id.toDouble())); load() } }, colors = ButtonDefaults.textButtonColors(textColor = MiuixTheme.colorScheme.error))
    }
   }
   Footnote("每天检查订阅更新；校验失败时保留上一份可用快照。")
   ActionButton("检查订阅更新", enabled && !loading, primary = false) { action { API.call("filter/update",json()); load() } }
   Group("规则管理") {
    SettingRow("自定义规则", "编辑黑名单和白名单", if (loading) "读取中" else "${array(if (data == null) null else data.user_rules).size} 条", onClick = { open("user-rules") })
    SettingRow("添加订阅", "添加其他 HTTPS 过滤规则", onClick = { open("add-filter") })
   }
  }
 }
}

@Composable internal fun Settings(rawState: String, enabled: Boolean, busy: Boolean, detail: String, open: (String) -> Unit, action: Action) {
 val state: dynamic = remember(rawState) { JSON.parse<dynamic>(rawState) }
 var ali by remember { mutableStateOf("") }; var cf by remember { mutableStateOf("") }; var boot by remember { mutableStateOf("") }
 var ipv6 by remember { mutableStateOf(true) }; var prefMode by remember { mutableStateOf("daily") }
 var password by remember { mutableStateOf("") }; var yaml by remember { mutableStateOf("") }; var confirm by remember { mutableStateOf("") }
 var error by remember { mutableStateOf("") }; var loading by remember { mutableStateOf(true) }
 LaunchedEffect(Unit) {
  try {
   val p = API.call("preferences"); val d = array(p.doh)
   ali = d.getOrNull(0)?.toString() ?: ""; cf = d.getOrNull(1)?.toString() ?: ""
   boot = array(p.bootstrap).joinToString("\n") { it.toString() }; ipv6 = bool(p,"ipv6"); prefMode = str(p,"mode","daily")
  } catch(e: Throwable) { error = e.message ?: "读取设置失败" }
  finally { loading = false }
 }
 if (error.isNotEmpty()) Footnote(error, color = MiuixTheme.colorScheme.error)
 when (detail) {
  "appearance" -> {
   Group("颜色模式") {
    ChoiceRow("跟随系统", "自动使用设备的浅色或深色外观", UiAppearance.mode == "system", true) { UiAppearance.select("system") }
    ChoiceRow("浅色", "浅灰背景与白色分组", UiAppearance.mode == "light", true) { UiAppearance.select("light") }
    ChoiceRow("深色", "深色背景与柔和的卡片层次", UiAppearance.mode == "dark", true) { UiAppearance.select("dark") }
   }
   Footnote("外观仅保存在当前浏览器，不影响 DNS、防护配置或设备授权。字体使用官方 MiSans VF。")
  }
  "session" -> {
   Group("当前会话") {
    SettingRow("管理账户", value = "root")
    SettingRow("操作权限", summary = if (API.authorized) "设备授权有效，可修改防护与配置" else "只读，可查看状态和记录", value = if (API.authorized) "已授权" else "只读")
   }
   FormGroup("设备授权") {
    Text("请在 Root 管理器的模块页面点击“操作”，再从打开的页面登录。设备授权用于修改配置，有效期为 5 分钟。", style = MiuixTheme.textStyles.body2)
    Text("授权过期后仍可查看状态；需要修改时再次从管理器打开。", style = MiuixTheme.textStyles.footnote1, color = MiuixTheme.colorScheme.onSurfaceVariantSummary)
   }
  }
  "dns" -> {
   val health = array(if (state == null) null else state.doh)
   FormGroup("公共加密 DNS") {
    Line("上游可用性", "上游一 ${if (health.getOrNull(0) == true) "可用" else "未知 / 失败"} · 上游二 ${if (health.getOrNull(1) == true) "可用" else "未知 / 失败"}")
    Input(ali,"公共 DoH 一",{ali = it}); Input(cf,"公共 DoH 二",{cf = it}); Input(boot,"Bootstrap DNS（每行一个 IP）",{boot = it})
   }
   Group("接管选项") {
    SwitchPreference(title = "尝试 IPv6 接管", summary = "能力不足时放行 IPv6 流量",
     checked = ipv6, onCheckedChange = { ipv6 = it }, enabled = enabled && !loading && error.isEmpty())
   }
   Footnote("任一公共 DoH 可用就优先使用；两者连续失败 3 次才回退到实际网络 DNS，全部失败则旁路。")
   ActionButton("保存 DNS 设置", enabled && !loading && error.isEmpty()) { action { API.call("preferences",json("schema" to 1,"mode" to str(state,"mode",prefMode),"doh" to arrayOf(ali,cf),"bootstrap" to boot.lines().filter {it.isNotBlank()}.toTypedArray(),"ipv6" to ipv6)) } }
  }
  "network" -> {
   val n = if (state == null) null else state.network
   Group("当前网络") {
    SettingRow("底层网络", if (state == null) "状态未知" else if (bool(n,"reliable")) str(n,"interface") else "无法可靠识别，安全旁路")
    SettingRow("系统 DNS",array(if (n == null) null else n.dns).joinToString().ifEmpty { if (state == null) "状态未知" else "未发现" })
    SettingRow("代理检测",if (state == null) "状态未知" else if (bool(n,"vpn")) "VPN / TUN 活跃，保留代理路径" else array(if (n == null) null else n.proxy).joinToString().ifEmpty { "未发现已知代理进程" })
   }
   FormGroup("私人 DNS") { Text(if (state == null) "状态未知" else if (bool(state,"privateDnsConflict")) "检测到用户更改，不会反复覆盖。" else "首次有效接管前备份并关闭；暂停期间保持关闭。",fontSize = 14.sp) }
   Footnote("root 发出的 DNS、应用内加密 DNS 和 VPN 内流量可能不被过滤。重新检查会尝试恢复防护，无法满足条件时保持旁路。")
   ActionButton("重新检查并尝试恢复",enabled) { action { API.call("resume",json()) } }
  }
  "security" -> {
   FormGroup("修改管理密码") {
    Line("用户名","root")
    Input(password,"新密码",{password = it},password = true)
    ActionButton("修改密码并退出",enabled && password.isNotEmpty()) { action { API.call("password",json("password" to password)); password = ""; API.logged = false; API.authorized = false } }
   }
   Footnote("默认 root / root 为公开凭据，本机其他 App 也可能尝试访问。修改后会退出登录，需要使用新密码重新登录。AGH 官方 API 与模块桥接共用同一核心。")
  }
  "backup" -> {
   FormGroup("导出配置") {
    Text("保存可移植 YAML，包含可移植设置与用户规则。完整原配置备份仅保存在 Root 数据目录。",fontSize = 14.sp)
    ActionButton("下载可移植配置",enabled,primary = false) { js("window").open("/api/backup","_blank") }
   }
   FormGroup("导入与恢复") {
    Input(yaml,"粘贴可移植 YAML",{yaml = it})
    Text("仅导入白名单字段，拒绝路径和监听配置。导入或恢复前请先导出备份。",fontSize = 13.sp,color = MiuixTheme.colorScheme.onSurfaceVariantSummary)
    Input(confirm,"输入“确认”以继续",{confirm = it})
    ActionButton("验证并导入",enabled && confirm == "确认" && yaml.isNotBlank()) { action { API.call("import",json("yaml" to yaml,"confirm" to true)); confirm = "" } }
    HorizontalDivider(color = MiuixTheme.colorScheme.onSurface.copy(alpha = 0.07f))
    Text("恢复过滤预设或整个配置会清除自定义规则和订阅；保留凭据与日志数据。",fontSize = 13.sp,color = MiuixTheme.colorScheme.error)
    listOf("恢复过滤预设" to "filters","恢复 DNS 设置" to "dns","恢复整个配置" to "all").forEach { (name,key) ->
     TextButton(name,modifier = Modifier.fillMaxWidth(),enabled = enabled && confirm == "确认",onClick = { action { API.call("restore",json("scope" to key,"confirm" to true)); confirm = "" } },colors = ButtonDefaults.textButtonColors(textColor = MiuixTheme.colorScheme.error))
    }
   }
  }
  "about" -> {
   val v = if (state == null) null else state.versions
   Spacer(Modifier.height(16.dp))
   IconTile(MiuixIcons.Filter,MiuixTheme.colorScheme.primary,large = true)
   Text("MyAdBlock",Modifier.padding(top = 16.dp),fontSize = 28.sp,fontWeight = FontWeight.Medium)
   Footnote("本机 DNS 广告过滤 · 网络可用优先")
   Group("版本信息") {
    SettingRow("模块版本",value = str(v,"moduleVersion"));
    SettingRow("WebUI 版本",value = str(v,"webuiVersion"));
    SettingRow("AdGuard Home",value = str(v,"aghVersion"));
    SettingRow("配置版本",value = str(v,"configSchemaVersion"));
    SettingRow("界面字体", "MiSans VF · 小米", "4.009")
   }
   Footnote("MiSans 使用小米官方原版字体，许可随模块附带。核心与模块仅支持手动更新，升级前请备份。")
   FormGroup("故障恢复") {
    Text("网页不可用时，在 Root 终端执行：",fontSize = 14.sp)
    Text("sh /data/adb/modules/myadblock/emergency.sh",fontSize = 13.sp)
    Text("也可在管理器停用模块并重启，以撤销本模块接管。",fontSize = 13.sp,color = MiuixTheme.colorScheme.onSurfaceVariantSummary)
   }
   ActionButton("打开官方备用 WebUI",primary = false,modifier = Modifier.padding(top = 16.dp)) {
    val p = str(if (state == null) null else state.ports,"agh","")
    if (p.toIntOrNull() != null) js("window").open("http://127.0.0.1:$p/","_blank","noopener,noreferrer")
   }
  }
  else -> {
   Group("个性化") {
    SettingRow("外观设置", "颜色模式与界面显示", UiAppearance.label(), onClick = { open("appearance") })
   }
   Group("网络与防护") {
    SettingRow("DNS 设置", "加密上游、Bootstrap 与 IPv6", onClick = { open("dns") })
    SettingRow("网络诊断", "接管覆盖、系统 DNS 与代理状态", onClick = { open("network") })
   }
   Group("配置与安全") {
    SettingRow("账户与安全", "修改本机管理密码", onClick = { open("security") })
    SettingRow("备份与恢复", "导出、导入或恢复配置", onClick = { open("backup") })
   }
   Group("更多") {
    SettingRow("管理会话", if (API.authorized) "设备授权有效" else "从 Root 管理器打开可授权修改", if (API.authorized) "已授权" else "只读", onClick = { open("session") })
    SettingRow("关于 MyAdBlock", "版本、字体与故障恢复", str(if (state == null) null else state.versions,"moduleVersion",""), onClick = { open("about") })
   }
   Group("") {
    BasicComponent(title = "退出登录", summary = "退出当前本机管理会话", enabled = !busy,
     titleColor = BasicComponentDefaults.titleColor(color = MiuixTheme.colorScheme.error),
     onClick = { action { API.call("logout", json()); API.logged = false; API.authorized = false } })
   }
  }
 }
}
