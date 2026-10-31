public struct RuntimeCatalog: Sendable, Equatable {
    public let node: [String]
    public let php: [String]
    public let python: [String]

    public init(node: [String], php: [String], python: [String]) {
        self.node = node
        self.php = php
        self.python = python
    }

    public static let bundled = RuntimeCatalog(
        node: ["24.18.0", "22.23.1", "20.20.2"],
        php: ["8.5.8", "8.4.23", "8.3.32"],
        python: ["3.13.14", "3.12.13", "3.11.15"]
    )
}
