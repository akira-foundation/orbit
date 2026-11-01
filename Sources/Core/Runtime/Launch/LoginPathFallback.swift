import Foundation

struct LoginPathFallback {
    let home: String?
    let fileManager: FileManager

    func path() -> String {
        var entries = ["/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin"]
        if let home {
            entries += [home + "/.bun/bin", home + "/.local/bin", home + "/.cargo/bin"]
            for root in [
                home + "/.nvm/versions/node", home + "/Library/Application Support/Herd/config/nvm/versions/node",
            ] {
                let versions = (try? fileManager.contentsOfDirectory(atPath: root)) ?? []
                entries += versions.sorted().map { root + "/" + $0 + "/bin" }
            }
        }
        return entries.joined(separator: ":")
    }
}
