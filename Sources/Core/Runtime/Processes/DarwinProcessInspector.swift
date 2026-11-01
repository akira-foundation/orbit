import Darwin

public struct DarwinProcessInspector: ProcessInspector {
    private let reader: any KernelProcessReading
    private let lsof: LsofRunner

    public init(reader: any KernelProcessReading = KernelProcessReader(), lsof: LsofRunner = LsofRunner()) {
        self.reader = reader
        self.lsof = lsof
    }

    public func identity(of pid: Int32) throws -> ProcessIdentity {
        try enrich(reader.snapshot(pid))
    }

    public func groupMembers(_ pgid: Int32) throws -> [ProcessIdentity] {
        try identities(reader.group(pgid))
    }

    public func descendants(of pid: Int32) throws -> [ProcessIdentity] {
        let processes: [KernelProcessSnapshot]
        do {
            processes = try reader.all()
        } catch {
            throw RuntimeError.message("inspect descendants of PID \(pid): \(error)")
        }
        var known: Set<Int32> = [pid]
        var descendants: [KernelProcessSnapshot] = []
        var changed = true
        while changed {
            changed = false
            for process in processes
            where process.pid > 0 && !known.contains(process.pid) && known.contains(process.ppid) {
                known.insert(process.pid)
                descendants.append(process)
                changed = true
            }
        }
        return try identities(descendants)
    }

    public func listenerOwners(_ endpoint: Endpoint) throws -> [ProcessIdentity] {
        guard let host = Self.normalizedIPv4(endpoint.host), (1...65535).contains(endpoint.port) else {
            throw RuntimeError.message("inspect listener: invalid IPv4 endpoint \(endpoint.host):\(endpoint.port)")
        }
        let output = try lsof.run(host: host, port: endpoint.port)
        let pids: [Int32]
        do {
            pids = try ListenerOutputParser.processIDs(output)
        } catch {
            throw RuntimeError.message("inspect listener \(host):\(endpoint.port): \(error)")
        }
        let snapshots = try pids.compactMap { pid -> KernelProcessSnapshot? in
            do {
                return try reader.snapshot(pid)
            } catch RuntimeError.processGone {
                return nil
            }
        }
        return try identities(snapshots)
    }

    public func signal(_ expected: ProcessIdentity, _ signal: ProcessSignal) throws {
        try verifyIdentity(expected)
        guard kill(expected.pid, signal.rawValue) == 0 else {
            let code = errno
            if code == ESRCH { throw RuntimeError.processGone }
            throw RuntimeError.message("signal PID \(expected.pid): \(String(cString: strerror(code)))")
        }
    }

    private func enrich(_ snapshot: KernelProcessSnapshot) throws -> ProcessIdentity {
        guard try reader.snapshot(snapshot.pid).isSameProcess(as: snapshot) else {
            throw RuntimeError.identityMismatch("changed before enrichment")
        }
        let arguments = try reader.arguments(snapshot.pid)
        guard try reader.snapshot(snapshot.pid).isSameProcess(as: snapshot) else {
            throw RuntimeError.identityMismatch("changed during enrichment")
        }
        return ProcessIdentity(
            pid: snapshot.pid,
            pgid: snapshot.pgid,
            startID: snapshot.startID,
            executable: snapshot.command,
            arguments: arguments
        )
    }

    private func identities(_ snapshots: [KernelProcessSnapshot]) throws -> [ProcessIdentity] {
        var identities: [ProcessIdentity] = []
        var prior: Int32 = 0
        for snapshot in snapshots.sorted(by: { $0.pid < $1.pid }) where snapshot.pid > 0 && snapshot.pid != prior {
            prior = snapshot.pid
            do {
                identities.append(try enrich(snapshot))
            } catch RuntimeError.processGone {
                continue
            }
        }
        return identities
    }

    private static func normalizedIPv4(_ host: String) -> String? {
        var address = in_addr()
        guard inet_pton(AF_INET, host, &address) == 1 else { return mappedIPv4(host) }
        var buffer = [CChar](repeating: 0, count: Int(INET_ADDRSTRLEN))
        guard inet_ntop(AF_INET, &address, &buffer, socklen_t(buffer.count)) != nil else { return nil }
        return String(decoding: buffer.prefix { $0 != 0 }.map { UInt8(bitPattern: $0) }, as: UTF8.self)
    }

    private static func mappedIPv4(_ host: String) -> String? {
        var address = in6_addr()
        guard inet_pton(AF_INET6, host, &address) == 1 else { return nil }
        let bytes = withUnsafeBytes(of: address) { Array($0) }
        guard bytes[0..<10].allSatisfy({ $0 == 0 }), bytes[10] == 0xff, bytes[11] == 0xff else { return nil }
        return bytes[12...].map(String.init).joined(separator: ".")
    }
}
