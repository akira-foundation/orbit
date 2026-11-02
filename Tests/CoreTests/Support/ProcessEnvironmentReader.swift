import Darwin

@testable import Core

enum ProcessEnvironmentReader {
    static func strings(of pid: Int32) throws -> [String] {
        var mib: [Int32] = [CTL_KERN, KERN_PROCARGS2, pid]
        var buffer = [UInt8](repeating: 0, count: 1 << 20)
        var size = buffer.count
        guard sysctl(&mib, UInt32(mib.count), &buffer, &size, nil, 0) == 0 else {
            throw RuntimeError.message("read arguments of PID \(pid): \(errno)")
        }
        return buffer[MemoryLayout<Int32>.size..<size].split(separator: 0).map { String(decoding: $0, as: UTF8.self) }
    }
}
