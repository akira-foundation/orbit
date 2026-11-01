import Darwin

public struct KernelProcessReader: KernelProcessReading {
    private static let attempts = 8

    public init() {}

    public func snapshot(_ pid: Int32) throws -> KernelProcessSnapshot {
        try Self.requireValid(pid)
        var process = kinfo_proc()
        var size = MemoryLayout<kinfo_proc>.stride
        var mib: [Int32] = [CTL_KERN, KERN_PROC, KERN_PROC_PID, pid]
        if sysctl(&mib, UInt32(mib.count), &process, &size, nil, 0) != 0 {
            let code = errno
            if Self.isGone(pid, code) { throw RuntimeError.processGone }
            throw RuntimeError.message("inspect PID \(pid): \(Self.describe(code))")
        }
        guard size == MemoryLayout<kinfo_proc>.stride else {
            if Self.isGone(pid, 0) { throw RuntimeError.processGone }
            throw RuntimeError.message("inspect PID \(pid): incomplete kernel identity")
        }
        guard !Self.isExiting(process) else { throw RuntimeError.processGone }
        let snapshot = Self.snapshot(of: process)
        guard snapshot.pid == pid, snapshot.pgid > 0 else {
            throw RuntimeError.message("inspect PID \(pid): incomplete kernel identity")
        }
        return snapshot
    }

    public func group(_ pgid: Int32) throws -> [KernelProcessSnapshot] {
        guard pgid > 0 else { throw RuntimeError.message("inspect process group: invalid PGID \(pgid)") }
        return try Self.list(
            [CTL_KERN, KERN_PROC, KERN_PROC_PGRP, pgid], context: "inspect process group \(pgid)"
        )
    }

    public func all() throws -> [KernelProcessSnapshot] {
        try Self.list([CTL_KERN, KERN_PROC, KERN_PROC_ALL], context: nil)
    }

    public func arguments(_ pid: Int32) throws -> String {
        try Self.requireValid(pid)
        let limit = try Self.argumentLimit(for: pid)
        var mib: [Int32] = [CTL_KERN, KERN_PROCARGS2, pid]
        for _ in 0..<Self.attempts {
            var buffer = [UInt8](repeating: 0, count: limit)
            var size = buffer.count
            if sysctl(&mib, UInt32(mib.count), &buffer, &size, nil, 0) == 0 {
                return try ProcessArgumentsParser.parse(Array(buffer.prefix(size)))
            }
            let code = errno
            if Self.isGone(pid, code) { throw RuntimeError.processGone }
            _ = try snapshot(pid)
            if code == EINVAL || code == EPERM { return "" }
            guard code == EIO else {
                throw RuntimeError.message("inspect arguments for PID \(pid): \(Self.describe(code))")
            }
            usleep(100)
        }
        throw RuntimeError.message("inspect arguments for PID \(pid): \(Self.describe(EIO))")
    }

    private static func argumentLimit(for pid: Int32) throws -> Int {
        var limit: Int32 = 0
        var size = MemoryLayout<Int32>.size
        var mib: [Int32] = [CTL_KERN, KERN_ARGMAX]
        guard sysctl(&mib, UInt32(mib.count), &limit, &size, nil, 0) == 0 else {
            throw RuntimeError.message("inspect arguments for PID \(pid): \(describe(errno))")
        }
        guard limit > 0 else {
            throw RuntimeError.message("inspect arguments for PID \(pid): invalid argument size limit \(limit)")
        }
        return Int(limit)
    }

    private static func list(_ mib: [Int32], context: String?) throws -> [KernelProcessSnapshot] {
        var mib = mib
        let stride = MemoryLayout<kinfo_proc>.stride
        for _ in 0..<attempts {
            var size = 0
            guard sysctl(&mib, UInt32(mib.count), nil, &size, nil, 0) == 0 else {
                throw failure(context, errno)
            }
            var processes = [kinfo_proc](repeating: kinfo_proc(), count: size / stride + size / stride / 8 + 1)
            size = processes.count * stride
            if sysctl(&mib, UInt32(mib.count), &processes, &size, nil, 0) != 0 {
                guard errno == ENOMEM else { throw failure(context, errno) }
                continue
            }
            return processes.prefix(size / stride).filter { !isExiting($0) }.map(snapshot(of:))
        }
        throw failure(context, ENOMEM)
    }

    private static func snapshot(of process: kinfo_proc) -> KernelProcessSnapshot {
        let command = withUnsafeBytes(of: process.kp_proc.p_comm) { bytes in
            String(decoding: bytes.prefix { $0 != 0 }, as: UTF8.self)
        }
        return KernelProcessSnapshot(
            pid: process.kp_proc.p_pid,
            pgid: process.kp_eproc.e_pgid,
            ppid: process.kp_eproc.e_ppid,
            startID: startID(process.kp_proc.p_un.__p_starttime),
            command: command
        )
    }

    private static func requireValid(_ pid: Int32) throws {
        guard pid > 0 else { throw RuntimeError.message("inspect process: invalid PID \(pid)") }
    }

    private static func failure(_ context: String?, _ code: Int32) -> RuntimeError {
        guard let context else { return RuntimeError.message(describe(code)) }
        return RuntimeError.message("\(context): \(describe(code))")
    }

    static func startID(_ start: timeval) -> String {
        let micros = String(start.tv_usec)
        return "\(start.tv_sec):" + String(repeating: "0", count: max(0, 6 - micros.count)) + micros
    }

    private static func isExiting(_ process: kinfo_proc) -> Bool {
        process.kp_proc.p_stat == CChar(SZOMB) || process.kp_proc.p_flag & P_WEXIT != 0
    }

    private static func isGone(_ pid: Int32, _ code: Int32) -> Bool {
        if code == ESRCH || code == ENOENT { return true }
        return kill(pid, 0) != 0 && errno == ESRCH
    }

    private static func describe(_ code: Int32) -> String {
        String(cString: strerror(code))
    }
}
