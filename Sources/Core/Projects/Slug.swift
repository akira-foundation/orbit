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
        let trimmed = result.trimmingCharacters(in: CharacterSet(charactersIn: "-."))
        return trimmed.isEmpty ? "project" : trimmed
    }

    private static func isAllowed(_ scalar: Unicode.Scalar) -> Bool {
        (scalar >= "a" && scalar <= "z") || (scalar >= "0" && scalar <= "9") || scalar == "."
    }
}
