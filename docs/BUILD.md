# 构建与个人预设

版本只在 version.properties 修改。普通发行加 build，重大更新加 major 并重置 build，新年 YY.1.1；本地编译不自动加号。

## 工具链

已验证开发机 macOS ARM64：Go 1.25.3、JDK 17/25、Gradle 9.8.1、Node 26.11.1、npm 11.20.0、Android SDK platform android-37.0 / build-tools 36.0.0。CI 使用 Linux SDK 同级平台，仍需实际跑过才算 CI 验证。

```sh
python3 tools/fetch.py
# 已安装其他 Go 位置时显式指定 GO；Go 模块优先国内镜像
GOPROXY=https://goproxy.cn .tools/go/bin/go mod download
webui/gradlew --no-daemon jsBrowserDistribution
AGH_TEST_BINARY="$PWD/.cache/darwin/AdGuardHome/AdGuardHome" tests/test.sh
python3 tools/build.py --skip-webui
python3 tools/audit.py dist/MyAdBlock-26.1.1-arm64.zip
```

网络代理需要启用且实际可达。下载脚本先取锁文件指定来源，官方包 SHA 与官方清单交叉核对；任意哈希变化中止，不静默升级。Go bootstrap 脚本仅为本次已验证 macOS ARM64 准备；其他平台请安装同版 Go 并显式设 GO。Gradle/npm 依赖在首次构建时获取，已经锁定及缓存后可用 --offline 重建；Android 安装/启动不下载任何运行资源。

Gradle 验证文件默认严格检查。仅在有意升级、人工核对来源后使用 --write-verification-metadata sha256，不能把自动接受新哈希当常规 CI。npm 锁文件带 integrity 并固定工具依赖；关闭额外的源码映射插件，避免 Webpack 多实例冲突，发布不含源码映射。MiSans VF 字体和三份快照纳入源码，不从变动的 URL 重取来重建。字体取自小米国内官网原始压缩包，未裁剪或修改，随附官方许可；build.py 在打包前核验字体 SHA-256。

```sh
# 独立交叉编译，可实际生成 ELF64 AArch64
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 .tools/go/bin/go build -trimpath -o build/myadblock ./cmd/myadblock
```

build.py 白名单取 module/、已编译二进制/DEX/WebUI、presets/、licenses/、docs/ 与发行元数据；不会递归打包整个工作目录。ZIP 固定时间、排序和权限，MANIFEST.json 包含逐文件 SHA-256。不要把 .cache、reference 或 private 推送到仓库。

## 个人包

```sh
mkdir -p private
# 把个人便携 YAML 保存为 private/personal.yaml
python3 tools/build.py --skip-webui --private-config private/personal.yaml
```

个人包带 -personal 后缀；仅首次配置创建时使用，与通用包同一控制代码。后续升级保留运行配置，不覆盖个人设置。支持的便携 YAML 例：

```yaml
schema_version: 34
dns:
  cache_size: 33554432
user_rules:
  - '@@||example.com^'
```

不支持把含任意路径/用户/监听地址的完整官方 YAML 直接导入。上游在 WebUI 类型化设置中修改。若便携 YAML 无效，首启拒绝启用并保留原网络；任务书的“任意完整 YAML”因此收紧为安全白名单，详见设计决策。

## 本机模拟

```sh
.tools/go/bin/go build -o build/myadblock-host ./cmd/myadblock
# --module 指向包含 webroot/、presets/、version.properties 的构建目录
build/myadblock-host daemon --simulate --module build/module --data .cache/simulation --agh .cache/darwin/AdGuardHome/AdGuardHome
```

模拟不执行 iptables/Android设置、网络发现或外部探测。端口见模拟数据目录 ports.json；控制器使用真实 macOS AGH。它可证明 API/页面运行，不能证明 Android 透明接管。

moduleVersionCode 由 moduleVersion 自动派生，无需另改数字。内核声明必须匹配锁文件，跳过前端编译时也校验前端构建版本，避免包装了旧核心/旧页面却显示新版本。
