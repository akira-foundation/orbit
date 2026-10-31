import Foundation

enum ManifestReader {
    static func package(in directory: String) throws -> PackageManifest? {
        let path = URL(fileURLWithPath: directory).appendingPathComponent("package.json").path
        guard let data = FileManager.default.contents(atPath: path) else {
            return nil
        }
        if hasTrailingComma(data) {
            throw AnalysisError.invalidManifest(path: path, reason: "trailing comma")
        }
        do {
            return try JSONDecoder().decode(PackageManifest.self, from: data)
        } catch {
            throw AnalysisError.invalidManifest(path: path, reason: String(describing: error))
        }
    }

    private static func hasTrailingComma(_ data: Data) -> Bool {
        var inString = false
        var escaped = false
        var pendingComma = false
        for byte in data {
            if inString {
                switch (escaped, byte) {
                case (true, _): escaped = false
                case (false, UInt8(ascii: "\\")): escaped = true
                case (false, UInt8(ascii: "\"")): inString = false
                default: break
                }
                continue
            }
            switch byte {
            case UInt8(ascii: " "), UInt8(ascii: "\n"), UInt8(ascii: "\r"), UInt8(ascii: "\t"):
                continue
            case UInt8(ascii: "}"), UInt8(ascii: "]"):
                if pendingComma {
                    return true
                }
            default:
                break
            }
            pendingComma = byte == UInt8(ascii: ",")
            inString = byte == UInt8(ascii: "\"")
        }
        return false
    }
}
