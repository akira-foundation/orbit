import Foundation

public enum PathList {
    public static func merge(_ first: String, _ second: String) -> String {
        var seen: Set<Substring> = []
        var merged: [Substring] = []
        for entry in (first + ":" + second).split(separator: ":") where seen.insert(entry).inserted {
            merged.append(entry)
        }
        return merged.joined(separator: ":")
    }

    public static func value(in environment: EnvironmentList) -> String {
        environment.value(of: "PATH") ?? ProcessInfo.processInfo.environment["PATH"] ?? ""
    }
}
