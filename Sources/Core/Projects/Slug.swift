import Foundation

public enum Slug {
    public static func make(from name: String) -> String {
        let folded = name.trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
            .folding(options: .diacriticInsensitive, locale: nil)
        let unscoped = folded.hasPrefix("@") ? String(folded.dropFirst()) : folded
        var result = ""
        var pendingSeparator = false
        for scalar in unscoped.unicodeScalars {
            if isAllowed(scalar) {
                if pendingSeparator, !result.isEmpty { result.append("-") }
                pendingSeparator = false
                result.unicodeScalars.append(scalar)
            } else {
                pendingSeparator = true
            }
        }
        let labels = result.split(separator: ".").compactMap(validLabel)
        return labels.isEmpty ? "project" : labels.joined(separator: ".")
    }

    private static let maxLabelLength = 63

    private static func validLabel(_ label: Substring) -> String? {
        let hyphens = CharacterSet(charactersIn: "-")
        let trimmed = String(label).trimmingCharacters(in: hyphens)
        let capped = String(trimmed.prefix(maxLabelLength)).trimmingCharacters(in: hyphens)
        return capped.isEmpty ? nil : capped
    }

    private static func isAllowed(_ scalar: Unicode.Scalar) -> Bool {
        (scalar >= "a" && scalar <= "z") || (scalar >= "0" && scalar <= "9") || scalar == "."
    }
}
