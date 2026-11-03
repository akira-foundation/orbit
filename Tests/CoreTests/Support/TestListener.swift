import Darwin

@testable import Core

struct TestListener {
    let descriptor: Int32
    let port: Int

    static func open() throws -> TestListener {
        let descriptor = socket(AF_INET, SOCK_STREAM, 0)
        guard descriptor >= 0 else { throw RuntimeError.message("socket: \(errno)") }
        _ = fcntl(descriptor, F_SETFD, FD_CLOEXEC)
        var address = sockaddr_in()
        address.sin_len = UInt8(MemoryLayout<sockaddr_in>.size)
        address.sin_family = sa_family_t(AF_INET)
        address.sin_addr.s_addr = inet_addr("127.0.0.1")
        var length = socklen_t(MemoryLayout<sockaddr_in>.size)
        let bound = withUnsafeMutablePointer(to: &address) { pointer in
            pointer.withMemoryRebound(to: sockaddr.self, capacity: 1) { generic in
                bind(descriptor, generic, length) == 0 && listen(descriptor, 1) == 0
                    && getsockname(descriptor, generic, &length) == 0
            }
        }
        guard bound else {
            Darwin.close(descriptor)
            throw RuntimeError.message("listen: \(errno)")
        }
        return TestListener(descriptor: descriptor, port: Int(UInt16(bigEndian: address.sin_port)))
    }

    func close() {
        Darwin.close(descriptor)
    }
}
