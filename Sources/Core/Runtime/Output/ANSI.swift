public enum ANSI {
    nonisolated(unsafe) private static let sequence = /\u{1B}\[[0-9;]*[A-Za-z]/.matchingSemantics(.unicodeScalar)

    public static func strip(_ text: String) -> String {
        guard text.unicodeScalars.contains("\u{1B}") else {
            return text
        }
        return text.replacing(sequence, with: "")
    }
}
