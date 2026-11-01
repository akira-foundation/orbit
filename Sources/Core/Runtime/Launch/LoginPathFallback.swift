import Foundation

struct LoginPathFallback {
    private let home: String?
    private let fileManager: FileManager

    init(home: String?, fileManager: FileManager) {
        self.home = home.flatMap(Self.cleaned)
        self.fileManager = fileManager
    }

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

    private static func cleaned(_ home: String) -> String? {
        let components = home.split(separator: "/")
        if components.isEmpty {
            return home.isEmpty ? nil : ""
        }
        return (home.hasPrefix("/") ? "/" : "") + components.joined(separator: "/")
    }
}
