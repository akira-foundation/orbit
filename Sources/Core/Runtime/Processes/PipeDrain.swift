import Darwin
import Foundation

enum PipeDrain {
    static func readToEnd(_ descriptors: [Int32]) -> [Data] {
        var captured = Array(repeating: Data(), count: descriptors.count)
        var open = Array(descriptors.indices)
        var buffer = [UInt8](repeating: 0, count: 16384)
        while !open.isEmpty {
            var polled = open.map { pollfd(fd: descriptors[$0], events: Int16(POLLIN), revents: 0) }
            guard poll(&polled, nfds_t(polled.count), -1) >= 0 || errno == EINTR else { break }
            open = open.enumerated().filter { slot, index in
                guard polled[slot].revents != 0 else { return true }
                let count = read(descriptors[index], &buffer, buffer.count)
                let failure = count < 0 ? errno : 0
                guard count <= 0 else {
                    captured[index].append(contentsOf: buffer[0..<count])
                    return true
                }
                return failure == EINTR || failure == EAGAIN
            }.map(\.element)
        }
        return captured
    }
}
