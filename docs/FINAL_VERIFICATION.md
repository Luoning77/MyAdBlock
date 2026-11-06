# 最终本地验证

本文件与最终 ZIP 同批生成，所有真机范围见 TEST_MATRIX.md。

- Go 单元、race 与 vet：通过；含官方 AGH 0.107.80 本地集成。
- Miuix：jsBrowserDistribution 实际构建成功；固定版本 0.9.4，Kotlin 2.4.20，Compose 1.12.0。
- 本机模拟浏览器：root/root 登录、设备授权、四页布局、离线中文、手机宽度与定时暂停实际状态可见；暂停时 AGH 过滤关闭、两族未接管。与真实 Android 透明接管没有等价关系。
- Java/D8：API 37、min API 29 编译成功，仅有弃用提示。
- Shell：所有模块入口 sh -n 通过。
- ARM64：官方核心与控制器均经 ELF64 AArch64 检查。
- 静态安全：未包含全局 DROP / flush、代理改写、飞行模式或其他 App 清理操作；公开包不含 reference/private/prompt/源码映射/设备 secret。
- ZIP：最终包包含逐文件 MANIFEST SHA-256 和外部 SHA-256 文件。重复构建比对的机器可读证据在 dist/reproducibility.json；为避免自引用，包哈希不写入包内文档。
- 许可证：GPL、Miuix、Go 及模块依赖、Skiko/Skia、MiSans 与三份规则许可附带；Gradle/npm/Go 校验输入保留在源码。
- CI 工作流已编写并锁定 Actions SHA，未远程执行；Linux namespace 真 NAT 专项当前未在 macOS 执行。
- Android 17 / KernelSU / Edge 以及 Magisk/APatch、真实网络/代理矩阵均未真机验证。

追加回归：受控改密的新旧凭据登录、磁盘 YAML 嵌套映射保留自定义设置、私人 DNS 写入失败不认领、预设补偿恢复原启用标记、订阅 API 失败回滚源文件/缓存且不取消用户暂停，均纳入最终测试。Skiko 的构建机 file URL 已改为运行时本机 URL，并重新打开浏览器验证加载。

MiSans VF 字体替换：官方原版 4.009，SHA-256 已锁定；离线 clean + jsBrowserDistribution 构建通过，本机模拟浏览器中文显示正常。原有 Noto 字体及许可已从当前构建输入移除，随包附 MiSans 官方许可；Android 真机仍待验证。

额外边界：首个 Bootstrap 不应答时第二个仍能解析；端口状态损坏不取消暂停；真实/模拟数据目录隔离；启动失败不误报进程存活。相关用例通过最终单元测试。规则批量写入后只发一次 AGH 刷新，避免每个订阅分别刷新触发限流；刷新失败受控恢复源文件和缓存，保留暂停状态。

HyperOS 界面改造：最终 jsBrowserDistribution 离线构建通过；四页、DNS/账户/备份/网络/关于/规则详情、日志筛选和返回均经本机模拟浏览器观察。只读暂停预览的七个时长、默认5分钟、滚动、禁用确认及遮罩退出已检查。未执行授权写入验证（自动审批拒绝一次性grant访问），未修改Android设备或真实网络接管。源码保留全部19个原有API端点；本批次不改后端，未重跑Go/race/真机测试，以上后端记录来自此前验证。历史截图已从公开源码与模块清理；当前本机 UI 证据及范围见 UI_REVIEW.md。

最终收口（2026-11-05）：仓库路径修正后，使用当前全部后端源码重新运行 tests/test.sh（含 AGH_TEST_BINARY 官方 0.107.80 集成、Go race、vet、Shell 与静态审计），全部通过；当前 Miuix 0.9.4 含 preference 模块的离线 jsBrowserDistribution 再次构建通过。以上不改变各项 Android 真机未验证结论，也不补充新版界面的授权写入验证。最终 ZIP 由同一锁定 WebUI 输出重复编译控制器、DEX 并打包，哈希结果见 dist/reproducibility.json。

InstallerX 整体 Miuix 收口：首页、日志、规则、设置、外观详情和暂停弹层采用同一系统。最终浅色警示正文使用 #C5392B，经独立复核解除唯一对比度P2并重拍首页；离线 jsBrowserDistribution 再次成功。手机浅／深色只读模拟检查范围和未验证项见 UI_REVIEW.md。本批 UI 验证未增加设备授权或执行 Android 操作。
