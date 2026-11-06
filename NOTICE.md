# MyAdBlock 第三方声明
MyAdBlock 独立实现，使用 GPL-3.0-or-later；完整文本见 LICENSE。参考模块只用于审计思路，无脚本、配置或个人数据被复制。

| 组件 | 固定版本 | 来源 | 许可 |
|---|---|---|---|
| AdGuard Home（未修改官方 ARM64 二进制） | 0.107.80 | https://github.com/AdguardTeam/AdGuardHome/tree/v0.107.80 | GPL-3.0 |
| Miuix UI/preference/icons/core/squircle/shader | 0.9.4 | https://github.com/compose-miuix-ui/miuix/tree/v0.9.4 | Apache-2.0 |
| Kotlin / Compose Multiplatform / coroutines | 2.4.20 / 1.12.0 / 1.11.0 | https://github.com/JetBrains | Apache-2.0 |
| Skiko / Skia | 0.150.1 / 其锁定上游 | https://github.com/JetBrains/skiko | Apache-2.0 / BSD 及第三方许可 |
| MiSans VF | 小米官方原版可变字体，固定版本及 SHA-256 见 dependencies.lock.json | https://hyperos.mi.com/font/zh/download/ | MiSans 字体知识产权许可协议（licenses/MiSans-License.pdf） |
| Go 与 x/crypto、net、sys、sync | go.mod/go.sum | https://go.dev / https://cs.opensource.google/go/x | BSD-3-Clause |
| miekg/dns | 1.1.68 | https://github.com/miekg/dns | BSD-3-Clause |
| go-yaml | v3.0.1 | https://github.com/go-yaml/yaml/tree/v3.0.1 | MIT / Apache-2.0 |
| Magisk 安装器 | v30.2 | https://github.com/topjohnwu/Magisk/blob/v30.2/scripts/module_installer.sh | GPL-3.0 |
| AdGuard DNS filter | 随包快照版本见文件头 | https://github.com/AdguardTeam/AdGuardSDNSFilter | GPL-3.0 |
| AWAvenue 日常/严格 | 1.7.8-release | https://github.com/TG-Twilight/AWAvenue-Ads-Rule | GPL-3.0 |

原始许可证见 licenses/ 与 presets/snapshots/。源码、构建输入与来源校验见 dependencies.lock.json、go.sum、Gradle verification-metadata.xml、npm package-lock.json。ARM64 核心归档 SHA-256 已对照官方 checksums.txt。快照与中文字体直接纳入源码，避免远程内容变动影响重建。

GPL 组件对应源码：AGH 上述精确标签及其 go.mod/go.sum、构建说明，Magisk 上述精确脚本标签；本项目源码随本地项目交付。发布者再分发二进制时应同时提供与发行包相匹配的完整源码及构建输入，不应只发布 ZIP。
