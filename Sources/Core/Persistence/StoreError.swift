public enum StoreError: Error, Equatable, Sendable {
    case projectNotFound(String)
    case groupNotFound(String)
}
