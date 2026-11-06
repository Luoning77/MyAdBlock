import java.util.Properties
plugins {
 kotlin("multiplatform") version "2.4.20"
 id("org.jetbrains.kotlin.plugin.compose") version "2.4.20"
 id("org.jetbrains.compose") version "1.12.0"
}
val projectVersions = Properties().apply { rootDir.parentFile.resolve("version.properties").inputStream().use(::load) }
version = projectVersions.getProperty("webuiVersion")
kotlin {
 js {
  browser { commonWebpackConfig { outputFileName = "myadblock.js" } }
  binaries.executable()
 }
 sourceSets {
  jsMain.dependencies {
   implementation(compose.runtime)
   implementation(compose.foundation)
   implementation(compose.animation)
   implementation(compose.ui)
   implementation(compose.components.resources)
   implementation("top.yukonga.miuix.kmp:miuix-ui:0.9.4")
   implementation("top.yukonga.miuix.kmp:miuix-icons:0.9.4")
   implementation("top.yukonga.miuix.kmp:miuix-preference:0.9.4")
   implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.11.0")
  }
 }
}
compose.resources { generateResClass = always }
rootProject.plugins.withType<org.jetbrains.kotlin.gradle.targets.js.nodejs.NodeJsPlugin> {
 rootProject.extensions.configure<org.jetbrains.kotlin.gradle.targets.js.nodejs.NodeJsEnvSpec> { download.set(false) }
}
