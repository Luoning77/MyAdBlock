# MyAdBlock
<!-- impeccable:product-schema 1 -->
## Platform
web
## Stack
任务书指定 Miuix Compose Multiplatform，采用 Kotlin/JS Web；本地 Go bridge 与官方 AdGuard Home 单核心。完全离线静态资源。
## Users
个人 Android Root 用户；目标设备 Android 17、ARM64、KernelSU，默认 Microsoft Edge（用户输入 egde，按上下文理解）。
## Product Purpose
通过 DNS 过滤减少广告；优先网络可用，随时暂停并撤销接管。
## Operating Context
模块管理器 action 打开 loopback 网页；首次无需 AGH 初始化。Magisk/APatch 属于待真机验证兼容范围。
## Capabilities and Constraints
四页：首页、查询日志、过滤规则、设置。日常/严格双预设。默认 root/root，额外设备授权用于写操作。实际接管与核心运行分别显示。不得捏造统计或兼容证据。
## Brand Commitments
Miuix/HyperOS 设置风格、跟随系统主题、底部导航及暂停弹窗。
## Product Principles
网络稳定优先；只撤销自己的改动；故障状态可见；用户配置升级保留；安全旁路不宣称外部网络可用。
