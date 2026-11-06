# MyAdBlock

为 Android ARM64 Root 设备提供本地 DNS 广告过滤，优先保持网络可用，随时暂停整个 DNS 接管。

🌐 [English](README_EN.md) | 简体中文

## 📖 项目状态

独立 Go 守护/安全 API + 官方 AdGuard Home 0.107.80 单核心 + 真正的 Miuix 0.9.4 Kotlin/JS WebUI。参考模块仅用于理解与审计，没有沿用其清理 App 数据、IPv6 DROP、代理改写或飞行模式操作。

已生成离线 ARM64 模块 ZIP，并在 macOS 完成编译、单元测试和真实官方 AGH 集成。**尚未在 Android 真机安装验证，不能视为已通过发行验收。** 目标设备是 Android 17 / KernelSU / Microsoft Edge；Magisk、APatch 采用通用入口，兼容性仍待验证。

## ✨ 功能

| 能力 | 实现与限制 |
|---|---|
| 双预设 | 日常 load_balance / 严格 parallel，AWAvenue 两版互斥，保留用户规则 |
| 公共 DoH 优先 | 阿里与 Cloudflare 为共同主要上游；任一可用仍公共，均失效 3 次才改用真实底层 DNS |
| 网络故障回退 | 系统 DNS 也失败、未知 VPN/NAT 冲突、内核能力不足时撤销自身接管 |
| 完整暂停 | 30 秒、1/5/10 分、1 小时、24 小时、无限期；停过滤及两族 NAT，定向清理自身 DNS conntrack |
| IPv4 / IPv6 | 每族能力探测和非 root canary 查询；不支持的族放行，不添加 DROP |
| 私人 DNS | 首次有效启用前备份并关闭；暂停仍关闭，永久退出/卸载按字段条件恢复 |
| 本地管理 | 开机随机 UI/API/DNS 端口，loopback 监听；action 打开 Miuix，单核心官方 UI 备用 |
| 安全桥接 | 会话、CSRF/Origin/Host 校验、管理器短期设备授权、固定类型 API，无任意 Shell |
| 离线安装 | 官方核心、三份完整快照、中文字体、JS/wasm 与许可证均在 ZIP 内 |
| 配置保护 | 原子写入、官方配置校验、last-good 备份、安全便携 YAML 导入及个人包流程 |

## 🚀 安装与使用

模块 ID 为 `myadblock`，属于独立新模块。已有旧 AdGuard Home 模块时，请先手动停用/卸载旧模块并重启，避免旧版接管规则冲突；不会自动迁移其私人配置。

环境门槛：Android 10+、ARM64、支持标准模块接口的 Root 管理器，以及支持 Canvas/WebAssembly 的现代浏览器。这些门槛不代表所有组合已经实测。

1. 从本地 `dist/` 取 `MyAdBlock-26.1.1-arm64.zip`，对照同目录 `.sha256`。本项目没有自动上传或发布 Release。
2. 在 KernelSU / Magisk / APatch 的模块页面刷入，重启。
3. 点击管理器的“操作”按钮，浏览器打开随机端口上的 Miuix 页面。
4. 用 **root / root** 登录。操作按钮生成一分钟单次授权，交换后可修改五分钟；超时重新点击操作按钮。
5. 分别查看“核心运行”“IPv4/IPv6 覆盖”“当前上游”。核心运行不等于所有流量已过滤；有异常会显示旁路原因。

**默认密码是公开的。随机端口与 loopback 不能阻止恶意本机 App 访问 AGH 原生 API。** 请在设置中修改 root 密码。修改会保留到后续升级，模块另有每设备随机内部控制凭据。已知默认密码不授予桥接的高危写权限，但不能消除原生 AGH 的残余风险。

## 🛟 暂停与故障恢复

暂停会撤销本模块的 IPv4/IPv6 接管，同时关闭 AGH 过滤；核心和网页继续运行，私人 DNS 保持关闭。到期通过上游与能力检查后才恢复，失败保持旁路。重启清除暂停。校园 Portal 可用暂停恢复原始网络路径；这不能保证校园网/上游本身一定可用。

网页不可用时，在 Root 终端执行：

```sh
sh /data/adb/modules/myadblock/emergency.sh
```

也可在管理器停用模块并重启。[故障恢复与卸载](docs/RECOVERY.md) 说明永久退出、配置回滚和私人 DNS 条件恢复。

## 🔧 构建和测试

```sh
python3 tools/fetch.py
webui/gradlew --no-daemon jsBrowserDistribution
AGH_TEST_BINARY="$PWD/.cache/darwin/AdGuardHome/AdGuardHome" tests/test.sh
python3 tools/build.py --skip-webui
```

详见[工具链与个人包](docs/BUILD.md)。版本唯一文件为 `version.properties`，本地构建不自动递增。Go、Miuix、Kotlin、Compose、规则快照和字体均固定；第三方校验在 `dependencies.lock.json`、`go.sum`、Gradle verification metadata 和 npm lock 中。CI 只构建/测试/上传工作流产物，不发布远程 Release。

## ⚠️ 覆盖边界

| 场景 | 行为 |
|---|---|
| FlClash/VPN/TUN、其他 NAT OUTPUT 规则 | 保守旁路并提示；不改代理 YAML/订阅，不切节点 |
| 系统网络 DNS 无法可靠发现 | 旁路，不伪造系统 DNS |
| IPv6 NAT / conntrack 不支持 | IPv6 放行或整体旁路，不全局阻断 |
| root 发出的 DNS、App 私有 DoH/DoT、VPN 内 DNS | 可能不被接管；无全覆盖承诺 |
| NFT-only / OEM 安全限制 / SELinux 拒绝 | 无强制绕过；旁路并诊断 |
| YAML 导入 | 接受安全便携子集，拒绝用户、路径、监听及任意字段；不是完整官方 YAML 导入器 |
| 二进制升级 | 只支持明确的重新构建/刷包，默认不在线替换核心或模块 |

[设计决策](docs/DESIGN_DECISIONS.md)、[测试矩阵](docs/TEST_MATRIX.md)、[分阶段交付](docs/DELIVERY.md) 逐项列出已测与未测。不要把模拟页面截图当成 Android 网络验收。

## 📱 界面

使用真实 Miuix 0.9.4 与离线 MiSans VF，参考 InstallerX-Revived 的整体 Miuix 体验。首页、查询日志、过滤规则、设置与弹层共用原生偏好组件、分组、文字和浅色／深色主题；详情页通过返回进入原入口。

[参考依据与实现边界](docs/UI_REFERENCE.md)、[本机界面核验](docs/UI_REVIEW.md)。本机模拟显示的安全旁路不代表 Android 真机接管验证。

## 🙏 致敬与鸣谢

MyAdBlock 在 [Luoning77/MyAdBlock](https://github.com/Luoning77/MyAdBlock) 独立开发和维护。向 [liuzq2002/Adguard-Home-For-Magisk-Mod](https://github.com/liuzq2002/Adguard-Home-For-Magisk-Mod) 致敬，感谢它为在 Root 模块中运行 AdGuard Home 提供的探索与启发。

原项目仅用于理解实现思路与安全审计。MyAdBlock 重新实现了控制层、网络回退、配置管理和 Miuix WebUI，没有移植原项目的 App 数据清理、IPv6 DROP、代理改写或飞行模式操作；审计记录见 [REFERENCE_AUDIT](docs/REFERENCE_AUDIT.md)。

感谢 [AdGuard Home](https://github.com/AdguardTeam/AdGuardHome)、[Miuix](https://github.com/compose-miuix-ui/miuix)、[AWAvenue](https://github.com/TG-Twilight/AWAvenue-Ads-Rule) 及其他依赖的维护者。

## 📄 许可

[GPL-3.0-or-later](LICENSE)。第三方组件与完整许可见 [NOTICE](NOTICE.md) 和 `licenses/`。
