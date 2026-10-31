struct PackageManifest: Decodable, Sendable {
    var name: String?
    var scripts: [String: String]?
    var dependencies: [String: String]?
    var devDependencies: [String: String]?
    var packageManager: String?
    var engines: [String: String]?

    var allDependencyNames: Set<String> {
        Set((dependencies ?? [:]).keys).union((devDependencies ?? [:]).keys)
    }
}
