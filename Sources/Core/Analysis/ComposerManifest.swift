struct ComposerManifest: Decodable, Sendable {
    var name: String?
    var require: [String: String]?

    var isLaravel: Bool {
        require?["laravel/framework"] != nil
    }

    var phpConstraint: String? {
        require?["php"]
    }
}
