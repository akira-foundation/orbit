public enum ANSI {
    private static let escape: Character = "\u{1B}"

    nonisolated(unsafe) private static let sequence = /\u{1B}\[[0-9;]*[A-Za-z]/

    public static func strip(_ text: String) -> String {
        guard text.contains(escape) else {
            return text
        }
        return text.replacing(sequence, with: "")
    }
}
