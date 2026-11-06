# InstallerX Miuix 参考与实现边界

主人明确指定 InstallerX-Revived 的整体 Miuix 体验，而不只是设置页面。参考固定到提交 `2bdd19dc552a3742b7b563dd9fa9a74ce56bb644`，避免将移动分支当固定设计依据。

## 一手实现依据

- [整体主题](https://github.com/wxxsfxyzm/InstallerX-Revived/blob/2bdd19dc552a3742b7b563dd9fa9a74ce56bb644/app/src/main/java/com/rosan/installer/ui/theme/InstallerTheme.kt)：Miuix ThemeController；支持默认主题及可选 Monet。
- [设置页](https://github.com/wxxsfxyzm/InstallerX-Revived/blob/2bdd19dc552a3742b7b563dd9fa9a74ce56bb644/app/src/main/java/com/rosan/installer/ui/page/miuix/settings/preferred/MiuixPreferredPage.kt)：SmallTitle、Card、BasicComponent；卡片左右12dp，卡片后12dp。
- [设置导航组件](https://github.com/wxxsfxyzm/InstallerX-Revived/blob/2bdd19dc552a3742b7b563dd9fa9a74ce56bb644/app/src/main/java/com/rosan/installer/ui/page/miuix/widgets/MiuixNavigationItemWidget.kt)：实际调用 ArrowPreference，没有将传入 icon 画成彩色方块。
- [首页](https://github.com/wxxsfxyzm/InstallerX-Revived/blob/2bdd19dc552a3742b7b563dd9fa9a74ce56bb644/app/src/main/java/com/rosan/installer/ui/page/miuix/settings/home/MiuixHomePage.kt)：状态卡与真实信息分组，状态卡16dp留白、20sp半粗标题、淡色背景图标。
- [导航和底栏](https://github.com/wxxsfxyzm/InstallerX-Revived/blob/2bdd19dc552a3742b7b563dd9fa9a74ce56bb644/app/src/main/java/com/rosan/installer/ui/page/miuix/settings/MiuixSettingsPage.kt)：标准与可选悬浮底栏、宽屏布局和可选原生模糊。

公开issue截图用于理解历史整体层次：[安装弹层](https://github.com/wxxsfxyzm/InstallerX-Revived/issues/329)、[配置页](https://github.com/wxxsfxyzm/InstallerX-Revived/issues/201)。截图年代和设备字体与上述代码不同，不作为精确字形或当前版本像素模板，截图文件仅留在忽略的 .cache 中，不放入安装包。

## 本项目采用

保留真实 Miuix 0.9.4，新增同版 miuix-preference Web依赖；使用 ArrowPreference、BasicComponent、RadioButtonPreference、SwitchPreference、SmallTitle及ThemeController。使用官方MiSans VF，12dp页面边距、16dp表单内缩，普通列表不增加彩色图标块或分隔线。

首页、日志、规则、设置、登录与弹层沿用同一主题、文字和卡片系统。标题与底栏保持库默认；详情隐藏底栏，通过浏览器history返回原入口。短距离进入过渡尊重 prefers-reduced-motion。外观模式仅保存于当前页面地址的浏览器localStorage，不涉及DNS或设备授权。

参考的Snapshot依赖、Android系统动态取色、原生触觉、原生模糊和预测返回不作为Web已实现能力。当前工作是让WebUI的视觉与可支持交互对齐此参考，不将本机模拟截图当成Android真机体验或网络验收。
