plugins {
    `java-library`
}

java {
    sourceSets.getByName("main").resources.srcDir(rootProject.file("build/v2ray-core"))
}
