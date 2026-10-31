public enum VersionConstraint {
    public static func resolve(_ constraint: String, among available: [String]) -> String? {
        let groups = constraint.split(separator: "|").compactMap { clauses(in: $0) }
        guard !groups.isEmpty else {
            return nil
        }
        let candidates = available.compactMap { raw in SemanticVersion(raw).map { (version: $0, raw: raw) } }
        let matches = candidates.filter { candidate in
            groups.contains { group in group.allSatisfy { $0.isSatisfied(by: candidate.version) } }
        }
        return matches.max { $0.version < $1.version }?.raw
    }

    private static func clauses(in group: Substring) -> [VersionClause]? {
        let terms = group.split(whereSeparator: \.isWhitespace)
        guard !terms.isEmpty else {
            return nil
        }
        let clauses = terms.compactMap { VersionClause(String($0)) }
        return clauses.count == terms.count ? clauses : nil
    }
}
