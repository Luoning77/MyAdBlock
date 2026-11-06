pluginManagement {
 repositories { maven("https://maven.aliyun.com/repository/gradle-plugin"); gradlePluginPortal(); mavenCentral(); google() }
}
dependencyResolutionManagement {
 repositories { maven("https://maven.aliyun.com/repository/public"); maven("https://maven.aliyun.com/repository/google"); mavenCentral(); google() }
}
rootProject.name = "MyAdBlockWebUI"
