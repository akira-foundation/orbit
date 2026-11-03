public enum OwnershipIdentifiers {
    public static func sessionID() -> String {
        randomHex(bytes: 8)
    }

    public static func token() -> String {
        randomHex(bytes: 16)
    }

    private static func randomHex(bytes count: Int) -> String {
        var generator = SystemRandomNumberGenerator()
        return (0..<count).map { _ in
            let digits = String(UInt8.random(in: .min ... .max, using: &generator), radix: 16)
            return digits.count == 1 ? "0" + digits : digits
        }.joined()
    }
}
