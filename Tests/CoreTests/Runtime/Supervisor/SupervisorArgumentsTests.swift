import Testing

@testable import Core

struct SupervisorArgumentsTests {
    private static let session = "0123456789abcdef"
    private static let token = String(repeating: "ab", count: 16)

    private static func argv(replacing options: [String]? = nil, command: [String] = ["/bin/true"]) -> [String] {
        let defaults = ["--session", session, "--token", token, "--control-fd", "3", "--cwd", "/tmp/project"]
        return ["--runtime-supervisor"] + (options ?? defaults) + ["--"] + command
    }

    @Test func parseAcceptsCompleteInvocation() throws {
        let argv = [
            "--runtime-supervisor", "--session", Self.session, "--token", Self.token, "--control-fd", "3", "--cwd",
            "/tmp/project", "--", "/bin/sh", "-c", "printf ready",
        ]
        let invocation = try SupervisorArguments.parse(argv)
        #expect(invocation.sessionID == Self.session && invocation.controlDescriptor == 3)
        #expect(invocation.token == Self.token)
        #expect(invocation.directory == "/tmp/project" && invocation.command == ["/bin/sh", "-c", "printf ready"])
        #expect(SupervisorArguments.make(invocation) == argv)
    }

    @Test(arguments: [
        (["--debug"], "runtime: supervisor mode missing"),
        (
            argv(replacing: ["--token", token, "--control-fd", "3", "--cwd", "/tmp/project"]),
            #"runtime: missing supervisor option "--session""#
        ),
        (
            argv(replacing: ["--session", session, "--token", token, "--token", token, "--cwd", "/tmp/project"]),
            #"runtime: duplicate or empty supervisor option "--token""#
        ),
        (
            argv(replacing: ["--session", session, "--token", token, "--control-fd", "stdin", "--cwd", "/tmp"]),
            "runtime: invalid supervisor control descriptor"
        ),
        (
            argv(replacing: ["--session", session, "--token", token, "--control-fd", "3", "--cwd", "project"]),
            "runtime: supervisor cwd must be absolute"
        ),
        (argv(command: []), "runtime: supervisor command missing"),
        (["--runtime-supervisor", "--session", session], "runtime: supervisor command missing"),
        (
            argv(replacing: ["--mystery", "value", "--session", session, "--token", token]),
            #"runtime: unknown supervisor option "--mystery""#
        ),
        (
            argv(replacing: ["--session", session, "--token", token, "--control-fd", "2", "--cwd", "/tmp"]),
            "runtime: invalid supervisor control descriptor"
        ),
        (
            argv(replacing: ["--session", "0123456789abcde", "--token", token, "--control-fd", "3", "--cwd", "/tmp"]),
            "runtime: invalid supervisor ownership identifiers"
        ),
        (
            argv(replacing: [
                "--session", session, "--token", "zz" + token.dropFirst(2), "--control-fd", "3", "--cwd", "/",
            ]),
            "runtime: invalid supervisor ownership identifiers"
        ),
        (argv(replacing: ["--session", session, "--token"]), "runtime: malformed supervisor options"),
        (
            argv(replacing: ["--session", "", "--token", token, "--control-fd", "3", "--cwd", "/tmp"]),
            #"runtime: duplicate or empty supervisor option "--session""#
        ),
    ])
    func parseRejectsMalformedInvocation(arguments: [String], message: String) {
        #expect(throws: RuntimeError.message(message)) {
            try SupervisorArguments.parse(arguments)
        }
    }

    @Test func parseAcceptsUppercaseHexIdentifiers() throws {
        let invocation = try SupervisorArguments.parse(
            Self.argv(replacing: [
                "--session", Self.session.uppercased(), "--token", Self.token.uppercased(), "--control-fd", "4",
                "--cwd", "/",
            ]))
        #expect(invocation.controlDescriptor == 4)
    }

    @Test func runIfRequestedIgnoresOtherArguments() {
        #expect(RuntimeSupervisor.runIfRequested(arguments: ["--debug"]) == nil)
        #expect(RuntimeSupervisor.runIfRequested(arguments: []) == nil)
    }

    @Test func ownershipIdentifiersAreLowercaseHexOfTheExpectedLength() throws {
        let session = OwnershipIdentifiers.sessionID()
        let token = OwnershipIdentifiers.token()
        #expect(session.count == 16 && token.count == 32)
        #expect((session + token).allSatisfy { "0123456789abcdef".contains($0) })
        #expect(OwnershipIdentifiers.token() != token)
        _ = try SupervisorArguments.parse(
            Self.argv(replacing: ["--session", session, "--token", token, "--control-fd", "3", "--cwd", "/"]))
    }
}
