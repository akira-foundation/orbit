enum SQLValue: Sendable, Equatable {
    case null
    case int(Int64)
    case double(Double)
    case text(String)

    var intValue: Int64? {
        switch self {
        case .int(let value): value
        case .double(let value): Int64(exactly: value.rounded(.towardZero))
        case .text(let value): Int64(value)
        case .null: nil
        }
    }

    var stringValue: String? {
        switch self {
        case .text(let value): value
        case .int(let value): String(value)
        case .double(let value): String(value)
        case .null: nil
        }
    }

    var doubleValue: Double? {
        switch self {
        case .double(let value): value
        case .int(let value): Double(value)
        case .text(let value): Double(value)
        case .null: nil
        }
    }
}

extension SQLValue {
    static func optional(_ value: String?) -> SQLValue {
        value.map(SQLValue.text) ?? .null
    }

    static func optional(_ value: Int?) -> SQLValue {
        value.map { .int(Int64($0)) } ?? .null
    }

    static func bool(_ value: Bool) -> SQLValue {
        .int(value ? 1 : 0)
    }
}
