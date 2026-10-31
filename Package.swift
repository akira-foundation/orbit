// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "Orbit",
    platforms: [.macOS("27.0")],
    products: [
        .executable(name: "Orbit", targets: ["Orbit"]),
        .executable(name: "orbit-agent", targets: ["orbit-agent"]),
        .executable(name: "orbit-helper", targets: ["orbit-helper"]),
        .executable(name: "orbit-mcp", targets: ["orbit-mcp"]),
        .library(name: "Core", targets: ["Core"]),
        .library(name: "OrbitDesign", targets: ["OrbitDesign"]),
    ],
    targets: [
        .target(
            name: "Core",
            swiftSettings: [.swiftLanguageMode(.v6)]
        ),
        .target(
            name: "OrbitDesign",
            dependencies: ["Core"],
            swiftSettings: [.swiftLanguageMode(.v6)]
        ),
        .executableTarget(
            name: "Orbit",
            dependencies: ["Core", "OrbitDesign"],
            swiftSettings: [.swiftLanguageMode(.v6)]
        ),
        .executableTarget(
            name: "orbit-agent",
            dependencies: ["Core"],
            swiftSettings: [.swiftLanguageMode(.v6)]
        ),
        .executableTarget(
            name: "orbit-helper",
            dependencies: ["Core"],
            swiftSettings: [.swiftLanguageMode(.v6)]
        ),
        .executableTarget(
            name: "orbit-mcp",
            dependencies: ["Core"],
            swiftSettings: [.swiftLanguageMode(.v6)]
        ),
        .testTarget(
            name: "CoreTests",
            dependencies: ["Core"],
            swiftSettings: [.swiftLanguageMode(.v6)]
        ),
        .testTarget(
            name: "OrbitDesignTests",
            dependencies: ["OrbitDesign"],
            swiftSettings: [.swiftLanguageMode(.v6)]
        ),
    ]
)
