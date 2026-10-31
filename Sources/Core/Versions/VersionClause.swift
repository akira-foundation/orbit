import Foundation

struct VersionClause: Sendable, Equatable {
    enum Operator: String, CaseIterable, Sendable {
        case caret = "^"
        case tilde = "~"
        case greaterOrEqual = ">="
        case lessOrEqual = "<="
        case greater = ">"
        case less = "<"
        case equal = "="
    }

    let comparison: Operator
    let version: SemanticVersion
    let isWildcard: Bool
    let significantParts: Int

    init?(_ term: String) {
        var text = term.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !text.isEmpty else {
            return nil
        }
        var comparison = Operator.equal
        if let matched = Operator.allCases.first(where: { text.hasPrefix($0.rawValue) }) {
            comparison = matched
            text.removeFirst(matched.rawValue.count)
        }
        text = text.trimmingCharacters(in: .whitespacesAndNewlines)
        let parts = text.split(separator: ".", omittingEmptySubsequences: false)
        let significant = parts.firstIndex { $0.isEmpty || $0 == "x" || $0 == "X" || $0 == "*" } ?? parts.count
        guard let version = SemanticVersion(text) else {
            return nil
        }
        self.comparison = comparison
        self.version = version
        self.significantParts = significant
        self.isWildcard = parts.count < 3 || significant < parts.count
    }

    func isSatisfied(by candidate: SemanticVersion) -> Bool {
        switch comparison {
        case .caret:
            return candidate.major == version.major && candidate >= version
        case .tilde:
            return candidate.major == version.major && candidate.minor == version.minor && candidate >= version
        case .greaterOrEqual:
            return candidate >= version
        case .lessOrEqual:
            return candidate <= version
        case .greater:
            return candidate > version
        case .less:
            return candidate < version
        case .equal:
            guard isWildcard else {
                return candidate == version
            }
            if significantParts >= 1 && candidate.major != version.major {
                return false
            }
            if significantParts >= 2 && candidate.minor != version.minor {
                return false
            }
            return true
        }
    }
}
