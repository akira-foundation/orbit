import Darwin
import Foundation
import Testing

@testable import Core

@Suite(.serialized)
struct DarwinListenerOwnersTests {
    @Test(arguments: [Endpoint(host: "::1", port: 80), Endpoint(host: "127.0.0.1", port: 0)])
    func listenerOwnersRejectsNonIPv4(endpoint: Endpoint) {
        let expected = RuntimeError.message(
            "inspect listener: invalid IPv4 endpoint \(endpoint.host):\(endpoint.port)"
        )
        #expect(throws: expected) {
            try DarwinProcessInspector().listenerOwners(endpoint)
        }
    }

    @Test func listenerOwnersFindsTheListeningProcess() throws {
        let listener = try TestListener.open()
        defer { listener.close() }
        let inspector = DarwinProcessInspector()
        let owners = try inspector.listenerOwners(Endpoint(host: "127.0.0.1", port: listener.port))
        #expect(owners == [try inspector.identity(of: getpid())])
    }

    @Test func listenerOwnersAcceptsAnIPv4MappedAddress() throws {
        let listener = try TestListener.open()
        defer { listener.close() }
        let inspector = DarwinProcessInspector()
        let owners = try inspector.listenerOwners(Endpoint(host: "::ffff:127.0.0.1", port: listener.port))
        #expect(owners == [try inspector.identity(of: getpid())])
    }

    @Test func listenerOwnersSkipAnOwnerThatVanished() throws {
        let directory = try TemporaryDirectory()
        try directory.write("#!/bin/sh\nprintf 'p10\\np12\\n'\n", to: "lsof")
        let path = directory.path + "/lsof"
        #expect(chmod(path, 0o755) == 0)
        let inspector = DarwinProcessInspector(
            reader: ScriptedKernelReader(table: [
                KernelProcessSnapshot(pid: 10, pgid: 10, ppid: 1, startID: "10:000001", command: "fixture")
            ]),
            lsof: LsofRunner(executable: URL(fileURLWithPath: path))
        )
        #expect(try inspector.listenerOwners(Endpoint(host: "127.0.0.1", port: 80)).map(\.pid) == [10])
    }

    @Test func listenerOwnersOfAClosedPortIsEmpty() throws {
        let listener = try TestListener.open()
        listener.close()
        let owners = try DarwinProcessInspector().listenerOwners(Endpoint(host: "127.0.0.1", port: listener.port))
        #expect(owners == [])
    }
}
