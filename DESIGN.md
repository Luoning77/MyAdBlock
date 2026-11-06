---
name: MyAdBlock
description: 对齐 InstallerX-Revived 的真实 Miuix 本机 DNS 管理界面
colors:
  light-primary: "#3482ff"
  dark-primary: "#277af7"
  light-error: "#c5392b"
  dark-error: "#f12522"
  light-error-container: "#fdf6f4"
  dark-error-container: "#2e0603"
  light-primary-container: "#5d9bff"
  dark-primary-container: "#338fe4"
  light-on-primary: "#ffffff"
  dark-on-primary: "#ffffff"
  light-surface: "#f7f7f7"
  dark-surface: "#000000"
  light-on-surface: "#000000"
  dark-on-surface: "#f2f2f2"
  light-surface-container: "#ffffff"
  dark-surface-container: "#242424"
  light-on-surface-container: "#000000"
  dark-on-surface-container: "#ffffffe6"
  light-summary: "#00000099"
  dark-summary: "#ffffff80"
  light-secondary-variant: "#f0f0f0"
  dark-secondary-variant: "#434343"
  light-on-secondary-variant: "#303030"
  dark-on-secondary-variant: "#d9d9d9"
  light-disabled-secondary-variant: "#f2f2f2"
  dark-disabled-secondary-variant: "#404040"
  light-disabled-on-secondary-variant: "#b2b2b2"
  dark-disabled-on-secondary-variant: "#707170"
  light-secondary-container: "#f0f0f0"
  dark-secondary-container: "#434343"
  light-on-secondary-container: "#a9a9a9"
  dark-on-secondary-container: "#7c7c7c"
typography:
  main:
    fontFamily: "MiSans VF"
    fontSize: "17px"
  paragraph:
    fontFamily: "MiSans VF"
    fontSize: "17px"
    lineHeight: 1.2
  body1:
    fontFamily: "MiSans VF"
    fontSize: "16px"
  body2:
    fontFamily: "MiSans VF"
    fontSize: "14px"
  button:
    fontFamily: "MiSans VF"
    fontSize: "17px"
  footnote1:
    fontFamily: "MiSans VF"
    fontSize: "13px"
  footnote2:
    fontFamily: "MiSans VF"
    fontSize: "11px"
  headline1:
    fontFamily: "MiSans VF"
    fontSize: "17px"
  headline2:
    fontFamily: "MiSans VF"
    fontSize: "16px"
  subtitle:
    fontFamily: "MiSans VF"
    fontSize: "14px"
    fontWeight: 700
  title1:
    fontFamily: "MiSans VF"
    fontSize: "32px"
  title2:
    fontFamily: "MiSans VF"
    fontSize: "24px"
  title3:
    fontFamily: "MiSans VF"
    fontSize: "20px"
  title4:
    fontFamily: "MiSans VF"
    fontSize: "18px"
  status:
    fontFamily: "MiSans VF"
    fontSize: "20px"
    fontWeight: 600
  note:
    fontFamily: "MiSans VF"
    fontSize: "12px"
rounded:
  miuix-control: "16px"
spacing:
  inline: "8px"
  stack: "12px"
  page-gutter: "12px"
  group-inset: "16px"
  page-bottom: "24px"
components:
  group-light:
    backgroundColor: "{colors.light-surface-container}"
    textColor: "{colors.light-on-surface-container}"
    rounded: "{rounded.miuix-control}"
    padding: "0px"
  group-dark:
    backgroundColor: "{colors.dark-surface-container}"
    textColor: "{colors.dark-on-surface-container}"
    rounded: "{rounded.miuix-control}"
    padding: "0px"
  form-light:
    backgroundColor: "{colors.light-surface-container}"
    rounded: "{rounded.miuix-control}"
    padding: "{spacing.group-inset}"
  primary-button-light:
    backgroundColor: "{colors.light-primary}"
    textColor: "{colors.light-on-primary}"
    typography: "{typography.button}"
    rounded: "{rounded.miuix-control}"
    padding: "13px 16px"
  text-button-light:
    backgroundColor: "{colors.light-secondary-variant}"
    textColor: "{colors.light-on-secondary-variant}"
    typography: "{typography.button}"
    rounded: "{rounded.miuix-control}"
    padding: "13px 16px"
  primary-button-dark:
    backgroundColor: "{colors.dark-primary}"
    textColor: "{colors.dark-on-primary}"
    typography: "{typography.button}"
    rounded: "{rounded.miuix-control}"
    padding: "13px 16px"
  text-button-dark:
    backgroundColor: "{colors.dark-secondary-variant}"
    textColor: "{colors.dark-on-secondary-variant}"
    typography: "{typography.button}"
    rounded: "{rounded.miuix-control}"
    padding: "13px 16px"
  input-light:
    backgroundColor: "{colors.light-secondary-container}"
    typography: "{typography.main}"
    rounded: "{rounded.miuix-control}"
  input-dark:
    backgroundColor: "{colors.dark-secondary-container}"
    typography: "{typography.main}"
    rounded: "{rounded.miuix-control}"
---
# Design System: MyAdBlock

## Overview

**Creative North Star: "InstallerX 式本机设置"**

整个 App 延续主人选定的 InstallerX-Revived Miuix 体验：首页、日志、规则、设置、登录和暂停弹层共用同一套主题、文字、卡片与操作控件。以克制的系统设置分组、清晰的标题和真实状态建立秩序，服务本机 DNS 管理与随时恢复网络的需求。

实现使用 Kotlin/JS Compose Canvas、真实 Miuix 0.9.4 及同版 miuix-preference。视觉参考固定为 InstallerX-Revived 提交 `2bdd19dc552a3742b7b563dd9fa9a74ce56bb644`；具体映射见 `docs/UI_REFERENCE.md`。应用源码是最终规范来源；sidecar 的 HTML/CSS 仅为可渲染的语义等效预览，不替代 Compose 实物。

**Key Characteristics:**
- 全 App 一致的 Miuix 系统设置语言。
- 紧凑单列、圆润分组、无装饰分隔线的普通列表。
- 首页淡语义状态底色与低透明度背景图标。
- 离线 MiSans、三种外观模式、支持减少动态效果。
- 旁路、未知、只读与故障必须清楚可见。

## Colors

基于 Miuix 默认浅深主题，用清爽的蓝色承载操作，以中性色承载信息；语义红仅用于错误、旁路与高影响操作。

### Primary
- **系统蓝**：`light-primary` / `dark-primary`，用于主要按钮、选中控件、通知及拦截统计数字。
- **淡蓝容器**：`light-primary-container` / `dark-primary-container`，正常或回退状态卡取该角色并覆盖透明度，不表示全部流量都已接管。

### Neutral
- **页面底色**：`light-surface` / `dark-surface`，浅灰与纯黑形成不同主题的底层。
- **分组表面**：`light-surface-container` / `dark-surface-container`，用明度分组，不给每行增加边框。
- **正文与说明**：`on-surface`、`on-surface-container` 和 `summary` 对应主题角色。说明使用带透明度的库颜色，按实际背景合成。
- **中性操作和输入表面**：`secondary-variant` / `secondary-container` 家族，保留禁用态角色；禁用控件同时禁用交互。

### Semantic
- **故障红**：浅色 `light-error` 是项目明确覆写，以改善故障文字对比；深色 `dark-error` 保留库值。`error-container` 仍使用库默认值，不能随错误文字色一并替换。
- 旁路或未知状态卡使用 `errorContainer` 的 45% 不透明度；正常或回退状态使用 `primaryContainer` 的 35% 不透明度；暂停使用普通分组表面。背景图标使用当前语义色的 10% 不透明度。

颜色只辅助说明；必须用文字分别陈述核心运行、接管覆盖、防护与故障，不能把蓝色或“操作已完成”当作接管证据。

## Typography

**Display Font:** MiSans VF。
**Body Font:** MiSans VF（小米官方原版 4.009）。

字体文件为 `webui/src/jsMain/composeResources/font/misans_vf.ttf`，许可随模块附带。保留库 `defaultTextStyles` 的字级与行高，仅统一字体；`preloadFont` 完成前不显示 Compose 界面。frontmatter 的 px 为面板预览用的名义尺寸，实际实现使用 sp，不能据此承诺系统字体缩放验收。

### Hierarchy
- **大标题**：库 `title1`，用于可随滚动收起的 TopAppBar。
- **状态标题**：`status`，半粗；说明使用 `body2`。
- **普通列表**：库 `headline1` 的中等字重标题，摘要用 `body2`；尾值为 14sp，最多两行并省略，最大宽 100dp。
- **正文与按钮**：`main`、`paragraph`、`button` 保留库尺寸；表单标签用 17sp、中等字重。
- **辅助说明**：`footnote1`；日志时间及状态使用 `note`。
- **统计**：32sp、中等字重；关于页名称 28sp，登录页标题 26sp，空态标题 20sp、中等字重。

## Layout

标题、正文与底部导航共同限宽 720dp，保持居中单列；正文两侧使用 `page-gutter`，顶部使用 `stack`，底部使用 `page-bottom`。这次已审阅的本机浏览器截图是 390×844，未据此确认桌面布局或 Android 真机表现。

分组顶部使用 `stack`，SmallTitle 水平内缩 16dp、垂直 8dp。Card 本身不统一增加留白，普通 BasicComponent / ArrowPreference 保留库的 16dp 内缩和至少 56dp 高度。FormGroup 使用 `group-inset` 与 16dp 纵向内容间距；状态卡同为 16dp 内缩、内容间距 10dp。统计卡两列、列间距 24dp。

四个主页面使用标准 NavigationBar，关闭分隔线。详情页隐藏底栏，标题显示返回按钮，浏览器 history 恢复原入口，包括从首页进入网络诊断的情况。切换页面与详情重新建立滚动状态，不承诺跨页滚动记忆。设置、规则同页内部详情切换保留当前页面草稿，离开对应主页面会重建状态。

页面进入与返回为方向相关的 18dp 横移，240ms，FastOutSlowInEasing，同时透明度从 0.85 到 1；检测到 `prefers-reduced-motion: reduce` 时直接显示终态。该偏好在 App 初始化时读取，不承诺运行期间监听切换。

## Elevation & Depth

以页面底色、卡片表面和语义浅底建立层次，不增加自定义卡片投影或渐变。标题收起、按钮 indication、底部弹层与遮罩沿用 Miuix。首页背景大图标是低透明度语义提示；不用于填补未知状态或掩盖故障。

真实产品使用 Miuix 的形状、状态与交互；不另写 HTML/CSS 仿制产品控件。sidecar 只服务设计面板预览。

## Shapes

Card、TextField 和 TextButton 使用库的圆润 squircle（`miuix-control`）。HTML/CSS 预览的 border-radius 只是近似曲线。普通设置和运行信息行不增加彩色图标块，也不增加逐行分隔线。登录与关于页仅使用裸图标，普通 24dp、大型 48dp；首页背景图标 136dp 并向右下偏移。

## Components

### Buttons

主操作使用 `textButtonColorsPrimary`，次操作使用中性 TextButton，保留库的最小尺寸（58×40dp）与内部留白。高影响操作用 error 文本；未授权或 busy 时禁用写入。选中的日志筛选使用主要按钮色，不另造 chip 样式。

### Cards / Containers

Group = SmallTitle + Card，FormGroup 在 Card 内统一留白。首页状态卡使用淡语义背景、标题、说明、实际故障原因和单一操作；普通分组表面承载统计及运行信息。Card 的圆角与裁切由库负责，不以截图近似值重写。

### Preferences

SettingRow：可进入详情时用 ArrowPreference，否则用 BasicComponent。普通行依靠标题、摘要、尾值和箭头组织，不添加图标块。单选使用末尾 RadioButtonPreference，适用于外观、规则预设与暂停时长；可选状态不能只靠禁用样式表达。IPv6 接管使用 SwitchPreference。

### Inputs / Fields

TextField 沿用库外观，标签清楚说明用途。长地址、密码、规则与 YAML 在详情中编辑；密码使用遮蔽输入。加载失败时不允许提交空草稿。恢复与导入的确认条件保持可见。

### Navigation

TopAppBar + MiuixScrollBehavior 保留库标题收起行为；主页面底栏显示图标与中文标签，详情返回原入口。“只读”按钮进入管理会话说明。外观页提供跟随系统、浅色、深色，使用 ThemeController；仅在当前 origin 的 localStorage 保存 `myadblock.appearance`，与防护配置及设备授权独立。

### Pause sheet

使用真实 OverlayBottomSheet。七个暂停时长，默认五分钟并标注常用，选中后仍需确认；关闭与遮罩退出不提交。内容高度按窗口计算、上限 560dp 并独立滚动，确认按钮位于选项区下方。只读可查看，选择与确认禁用，并说明设备授权要求。

### Feedback / Empty states

未知统计显示破折号与说明；空日志以中性图标、标题和操作提示组成空态。连接失败、操作失败、授权变化和 busy 分开处理；错误保持可见，成功短通知只说明请求完成。不得添加展示用统计、兼容性证据或虚构网络结果。

## Do's and Don'ts

### Do:
- **Do** 将整个 App 的新页面继续建立在同版 Miuix 组件、MiSans 与三种外观模式上。
- **Do** 保持紧凑页边距、无逐行分隔线的圆润分组与真实语义状态。
- **Do** 为只读、加载、空态、失联和错误保留明确文字与可用退出路径。
- **Do** 将源码视为规范来源，并区分设计面板近似预览与 Compose 实物。

### Don't:
- **Don't** 恢复普通列表的彩色图标块、旧宽间距或详情页底栏。
- **Don't** 用样式、演示数据或成功通知暗示真实接管、网络可用或完整覆盖。
- **Don't** 承诺 Android 原生触觉、系统动态取色、原生模糊或预测返回。
- **Don't** 把 390×844 本机只读安全旁路模拟截图当作桌面、Android 真机、字体缩放、精确对比度或读屏验收。
