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
        return VersionConstraint.resolve(constraint, among: catalog.node)
            ?? closest(to: constraint, among: catalog.node, matchingMinor: false) ?? newest
    }

    func php(constraint: String?) -> String? {
        guard let newest = catalog.php.first else {
            return nil
        }
        guard let constraint, !constraint.isEmpty else {
            return newest
        }
        return VersionConstraint.resolve(constraint, among: catalog.php)
            ?? closest(to: constraint, among: catalog.php, matchingMinor: true)
            ?? closest(to: constraint, among: catalog.php, matchingMinor: false) ?? newest
    }

    func python(in directory: String) -> String? {
        guard let newest = catalog.python.first else {
            return nil
        }
        let constraint = versionFile(".python-version", in: directory) ?? requiresPython(in: directory)
        guard let constraint else {
            return newest
        }
        if let resolved = VersionConstraint.resolve(constraint, among: catalog.python) {
            return resolved
        }
        return catalog.python.first { version in
            guard let dot = version.lastIndex(of: ".") else {
                return false
            }
            return constraint.contains(version[..<dot])
        } ?? newest
    }

    private func closest(to constraint: String, among available: [String], matchingMinor: Bool) -> String? {
        guard let wanted = SemanticVersion(constraint) else {
            return nil
        }
        let candidates = available.compactMap { raw in SemanticVersion(raw).map { (version: $0, raw: raw) } }
        return candidates.filter { candidate in
            candidate.version.major == wanted.major && (!matchingMinor || candidate.version.minor == wanted.minor)
        }
        .max { $0.version < $1.version }?.raw
    }

    private func requiresPython(in directory: String) -> String? {
        let path = URL(fileURLWithPath: directory).appendingPathComponent("pyproject.toml").path
        guard let contents = try? String(contentsOfFile: path, encoding: .utf8) else {
            return nil
        }
        for rawLine in contents.split(separator: "\n", omittingEmptySubsequences: false) {
            let line = rawLine.trimmingCharacters(in: .whitespaces)
            guard
                line.hasPrefix("requires-python") || line.hasPrefix("python ") || line.hasPrefix("python="),
                let equals = line.firstIndex(of: "=")
            else {
                continue
            }
            let value = line[line.index(after: equals)...]
                .trimmingCharacters(in: .whitespaces)
                .trimmingCharacters(in: CharacterSet(charactersIn: "\"'"))
            return value.isEmpty ? nil : value
        }
        return nil
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
