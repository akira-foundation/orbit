import Foundation

public struct SemanticVersion: Sendable, Hashable, Comparable, CustomStringConvertible {
    public let major: Int
    public let minor: Int
    public let patch: Int

    public init(_ major: Int, _ minor: Int, _ patch: Int) {
        self.major = major
        self.minor = minor
        self.patch = patch
    }

    public init?(_ string: String) {
        var text = string.trimmingCharacters(in: .whitespacesAndNewlines)
        if text.hasPrefix("v") {
            text.removeFirst()
        }
        var numbers: [Int] = []
        for part in text.split(separator: ".", maxSplits: 2, omittingEmptySubsequences: false) {
            if part.isEmpty || part == "x" || part == "X" || part == "*" {
                break
            }
            guard let number = Int(part) else {
                return nil
            }
            numbers.append(number)
        }
        guard let major = numbers.first else {
            return nil
        }
        self.init(major, numbers.count > 1 ? numbers[1] : 0, numbers.count > 2 ? numbers[2] : 0)
    }

    public var description: String {
        "\(major).\(minor).\(patch)"
    }

    public static func < (lhs: SemanticVersion, rhs: SemanticVersion) -> Bool {
        (lhs.major, lhs.minor, lhs.patch) < (rhs.major, rhs.minor, rhs.patch)
    }
}
