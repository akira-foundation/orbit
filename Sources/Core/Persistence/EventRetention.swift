public enum EventRetention {
    public static let maximumAge: Duration = .seconds(14 * 86_400)
    public static let maximumProjectBytes: Int64 = 256 * 1024 * 1024
    public static let interval: Duration = .seconds(3600)
    static let ageInSeconds = Double(maximumAge.components.seconds)
    static let batchRows = 1000
    static let vacuumPages = 256
    static let rowOverheadBytes: Int64 = 180
}
