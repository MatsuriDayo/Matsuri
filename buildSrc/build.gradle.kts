plugins {
    kotlin("jvm") version "1.5.31"
    `java-gradle-plugin`
    `kotlin-dsl`
}

apply(from = "../repositories.gradle.kts")

dependencies {
    // Gradle Plugins
    implementation("com.android.tools.build:gradle:7.3.1")
    implementation("org.jetbrains.kotlin:kotlin-gradle-plugin:1.6.21")
    //
    implementation("org.tukaani:xz:1.9")
    implementation("org.kohsuke:github-api:1.131")
    implementation("com.squareup.okhttp3:okhttp:5.0.0-alpha.3")
}
