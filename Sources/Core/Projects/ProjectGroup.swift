import Foundation

public struct ProjectGroup: Sendable, Equatable, Identifiable {
    public var id: String
    public var name: String
    public var sortOrder: Int
    public var createdAt: Date
    public var projectIDs: [String]

    public init(id: String, name: String, sortOrder: Int, createdAt: Date, projectIDs: [String] = []) {
        self.id = id
        self.name = name
        self.sortOrder = sortOrder
        self.createdAt = createdAt
        self.projectIDs = projectIDs
    }
}
