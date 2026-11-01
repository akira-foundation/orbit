public enum CommandLineSplitter {
    public static func split(_ line: String) -> [String] {
        var tokens: [String] = []
        var current = ""
        var quoted = false
        for character in line {
            if character == "\"" {
                quoted.toggle()
            } else if character == " " && !quoted {
                if !current.isEmpty {
                    tokens.append(current)
                    current = ""
                }
            } else {
                current.append(character)
            }
        }
        if !current.isEmpty {
            tokens.append(current)
        }
        return tokens
    }
}
