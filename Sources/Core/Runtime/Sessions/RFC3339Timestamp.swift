import Foundation

enum RFC3339Timestamp {
    static func seconds(_ date: Date) -> String {
        date.formatted(.iso8601)
    }

    static func nanoseconds(_ date: Date) -> String {
        let microseconds = Int64((date.timeIntervalSince1970 * 1_000_000).rounded())
        let (whole, fraction) = floorDivision(microseconds, by: 1_000_000)
        let base = seconds(Date(timeIntervalSince1970: TimeInterval(whole))).dropLast()
        guard fraction != 0 else { return "\(base)Z" }
        var digits = String(fraction)
        digits = String(repeating: "0", count: 6 - digits.count) + digits
        while digits.hasSuffix("0") { digits.removeLast() }
        return "\(base).\(digits)Z"
    }

    static func date(_ text: String) -> Date? {
        guard let dot = text.firstIndex(of: "."), text.hasSuffix("Z") else {
            return try? Date(text, strategy: .iso8601)
        }
        let digits = text[text.index(after: dot)..<text.index(before: text.endIndex)]
        guard !digits.isEmpty, digits.allSatisfy(("0"..."9").contains), let fraction = Double("0.\(digits)"),
            let whole = try? Date(text[..<dot] + "Z", strategy: .iso8601)
        else { return nil }
        return whole.addingTimeInterval(fraction)
    }

    static func decode<Key: CodingKey>(_ container: KeyedDecodingContainer<Key>, _ key: Key) throws -> Date {
        let text = try container.decode(String.self, forKey: key)
        guard let date = date(text) else {
            throw DecodingError.dataCorruptedError(
                forKey: key, in: container, debugDescription: "expected an RFC 3339 date, got \(text)")
        }
        return date
    }

    private static func floorDivision(_ value: Int64, by divisor: Int64) -> (Int64, Int64) {
        let remainder = value % divisor
        guard remainder < 0 else { return (value / divisor, remainder) }
        return (value / divisor - 1, remainder + divisor)
    }
}
