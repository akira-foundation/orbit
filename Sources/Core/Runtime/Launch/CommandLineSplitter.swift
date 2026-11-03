import Foundation

public enum CommandLineSplitter {
    public static func split(_ line: String) -> [String] {
        var tokens: [String] = []
        var current = String.UnicodeScalarView()
        var quoted = false
        for scalar in line.trimmingCharacters(in: .whitespacesAndNewlines).unicodeScalars {
            if scalar == "\"" {
                quoted.toggle()
                continue
            }
            guard scalar == " " && !quoted else {
                current.append(scalar)
                continue
            }
            if !current.isEmpty {
                tokens.append(String(current))
                current = String.UnicodeScalarView()
            }
        }
        if !current.isEmpty {
            tokens.append(String(current))
        }
        return tokens
    }
}
