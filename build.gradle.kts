// Top-level build file where you can add configuration options common to all sub-projects/modules.
allprojects {
    apply(from = "${rootProject.projectDir}/repositories.gradle.kts")
}

tasks.register<Delete>("clean") {
    delete(rootProject.buildDir)
}

subprojects {
    // skip uploading the mapping to Crashlytics
    tasks.whenTaskAdded {
        if (name.contains("uploadCrashlyticsMappingFile")) enabled = false
    }
}

tasks.named<Wrapper>("wrapper") {
    doLast {
        val sha256 = java.net.URL("$distributionUrl.sha256")
            .openStream()
            .use { it.reader().readText().trim() }

        file("gradle/wrapper/gradle-wrapper.properties").appendText("distributionSha256Sum=$sha256")
    }
}