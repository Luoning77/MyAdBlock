# MyAdBlock

Local DNS filtering for rooted ARM64 Android devices, with network availability and complete capture suspension as priorities.

🌐 English | [简体中文](README.md)

## Status

Independent Go controller/API bridge, one unmodified official AdGuard Home 0.107.80 core, and genuine Miuix 0.9.4 Kotlin/JS UI. An offline installable module ZIP is built locally. Unit tests and integration with the official macOS ARM64 AGH core pass. **Android installation and network interception have not been device-tested.** Target: Android 17, KernelSU, Microsoft Edge. Standard Magisk/APatch entry points are implemented, with compatibility pending real devices.

## Features

| Feature | Behavior |
|---|---|
| Presets | Daily load_balance / strict parallel; mutually exclusive AWAvenue lists; user rules preserved |
| Upstreams | AliDNS and Cloudflare DoH share primary priority; system DNS only after both fail three probes |
| Fail-safe | Remove only module-owned NAT rules on upstream/core/capability/proxy conflicts |
| Pause | 30 seconds, 1/5/10 minutes, 1 hour, 24 hours, indefinite; disable filtering, remove IPv4/IPv6 capture and its conntrack entries |
| Offline UI | Genuine Miuix four-page UI, local JS/Skiko wasm and Chinese font, no CDN runtime dependency |
| Security | Loopback listeners, random ports, sessions, CSRF/Origin/Host validation, short-lived per-device write grants, no shell API |
| Configuration | Atomic validated writes, last-good backup, bounded portable YAML import and explicit private builds |

## Install

Use the ZIP and matching SHA-256 in `dist/`, install through your root manager, then reboot. Use its action button to open the current local Miuix URL. Android 10+, ARM64 and a modern Canvas/WebAssembly browser are required; these requirements do not establish tested compatibility.

Login: **root / root**, as explicitly required. The action button supplies a one-minute single-use grant that enables writes for five minutes. Reopen through the manager when it expires.

**The default password is public. Loopback and random ports cannot protect the native AGH API from malicious local apps. Change the root password in Settings.** Bridge write grants add protection but do not remove the native API's default-password risk.

## Recovery

```sh
sh /data/adb/modules/myadblock/emergency.sh
```

Pause leaves AGH and the UI running, disables filtering and removes this module's capture. Private DNS remains off during pause. Resume requires healthy upstreams and verified capture. A reboot clears pause; permanent exit/uninstall conditionally restores only private-DNS fields still matching this module's writes.

## Build

```sh
python3 tools/fetch.py
webui/gradlew --no-daemon jsBrowserDistribution
AGH_TEST_BINARY="$PWD/.cache/darwin/AdGuardHome/AdGuardHome" tests/test.sh
python3 tools/build.py --skip-webui
```

See [build instructions](docs/BUILD.md), [decisions and limitations](docs/DESIGN_DECISIONS.md), [test matrix](docs/TEST_MATRIX.md), and [recovery](docs/RECOVERY.md). `version.properties` is the only project-version source. Dependencies and checksums are locked. CI builds artifacts without publishing releases.

## Limits

| Scenario | Policy |
|---|---|
| VPN/TUN or foreign OUTPUT NAT | Bypass; never rewrite proxy files or toggle airplane mode |
| Unreliable network DNS discovery | Bypass rather than invent addresses |
| Unsupported IPv6 NAT / targeted conntrack | Leave unsupported traffic untouched or bypass |
| Root DNS / private encrypted DNS / VPN traffic | Coverage not guaranteed |
| YAML import | Portable allowlisted subset only; listeners, paths and users rejected |
| Binary updates | Explicit rebuild/module reinstall; no automatic online replacement |

The interface uses Miuix 0.9.4 and Xiaomi’s original offline MiSans VF font. Home, query logs, filtering rules, settings and overlays share InstallerX-inspired preference components, consistent grouping and light/dark appearance. See [reference and boundaries](docs/UI_REFERENCE.md) and [local UI verification](docs/UI_REVIEW.md). Local simulation is not Android network validation.

## Acknowledgements

MyAdBlock is independently developed and maintained at [Luoning77/MyAdBlock](https://github.com/Luoning77/MyAdBlock). We pay tribute to [liuzq2002/Adguard-Home-For-Magisk-Mod](https://github.com/liuzq2002/Adguard-Home-For-Magisk-Mod) for its exploration of running AdGuard Home as a root module.

The reference project was studied for ideas and safety auditing. MyAdBlock implements its own controller, network fallback, configuration management and Miuix WebUI. No app-data cleanup, IPv6 DROP, proxy rewriting or airplane-mode operations were copied. See the [reference audit](docs/REFERENCE_AUDIT.md).

Thanks to [AdGuard Home](https://github.com/AdguardTeam/AdGuardHome), [Miuix](https://github.com/compose-miuix-ui/miuix), [AWAvenue](https://github.com/TG-Twilight/AWAvenue-Ads-Rule), and the maintainers of the other dependencies.

## License

[GPL-3.0-or-later](LICENSE); full third-party notices in [NOTICE](NOTICE.md) and `licenses/`.
