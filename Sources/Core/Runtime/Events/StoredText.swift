enum StoredText {
    static let maximumBytes = 64 * 1024
    static let marker = " [truncated]"

    static func limited(_ text: String) -> String {
        guard text.utf8.count > maximumBytes else { return text }
        let utf8 = text.utf8
        var end = utf8.index(utf8.startIndex, offsetBy: maximumBytes - marker.utf8.count)
        while end > utf8.startIndex, utf8[end] & 0xC0 == 0x80 {
            end = utf8.index(before: end)
        }
        return String(text[..<end]) + marker
    }
}
