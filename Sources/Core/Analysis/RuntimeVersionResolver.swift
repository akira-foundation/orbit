import Foundation

struct RuntimeVersionResolver: Sendable {
    let catalog: RuntimeCatalog

    func node(in directory: String, engines: String?) -> String? {
        guard let newest = catalog.node.first else {
            return nil
        }
        let constraint =
            versionFile(".nvmrc", in: directory)
            ?? versionFile(".node-version", in: directory)
            ?? engines.flatMap { $0.isEmpty ? nil : $0 }
        guard let constraint else {
            return newest
        }
        return VersionConstraint.resolve(constraint, among: catalog.node) ?? newest
    }

    func php(constraint: String?) -> String? {
        guard let newest = catalog.php.first else {
            return nil
        }
        guard let constraint, !constraint.isEmpty else {
            return newest
        }
        return VersionConstraint.resolve(constraint, among: catalog.php) ?? newest
    }

    private func versionFile(_ name: String, in directory: String) -> String? {
        let path = URL(fileURLWithPath: directory).appendingPathComponent(name).path
        guard let contents = try? String(contentsOfFile: path, encoding: .utf8) else {
            return nil
        }
        let line = contents.split(separator: "\n", maxSplits: 1, omittingEmptySubsequences: false).first ?? ""
        var trimmed = line.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.hasPrefix("v") {
            trimmed.removeFirst()
        }
        return trimmed.isEmpty ? nil : trimmed
    }
}
