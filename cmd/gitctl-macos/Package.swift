// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "GitCtl",
    platforms: [.macOS(.v14)],
    targets: [
        .executableTarget(
            name: "GitCtl",
            path: ".",
            exclude: ["Package.swift", "Info.plist"],
            swiftSettings: [
                .unsafeFlags(["-parse-as-library"]),
            ]
        ),
    ]
)
