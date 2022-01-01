rootProject.extra.apply {
    set("androidPluginVersion", "7.1.3")
    set("kotlinVersion", "1.6.21")
}

repositories {
    google()
    mavenCentral()
    gradlePluginPortal()
    maven(url = "https://jitpack.io")
}