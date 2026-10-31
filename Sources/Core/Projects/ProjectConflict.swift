public struct ProjectConflict: Error, Equatable, Sendable {
    public let existingName: String
    public let slug: String

    public init(existingName: String, slug: String) {
        self.existingName = existingName
        self.slug = slug
    }
}
