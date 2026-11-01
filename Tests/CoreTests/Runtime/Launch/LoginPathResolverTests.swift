import Foundation
import Testing

@testable import Core

struct LoginPathResolverTests {
    @Test func usesTheFirstMarkerLineAndRunsOnce() async {
        let runner = StubShell(output: "motd\n__ORBIT_PATH__:/a:/b\n__ORBIT_PATH__:/c\n")
        let resolver = LoginPathResolver(runner: runner, shell: "/bin/zsh", home: nil)
        #expect(await resolver.path() == "/a:/b")
        #expect(await resolver.path() == "/a:/b")
        #expect(runner.calls == [.init(shell: "/bin/zsh", script: "echo \(LoginPathResolver.marker)$PATH")])
    }

    @Test func missingShellDefaultsToZsh() async {
        let runner = StubShell(output: "__ORBIT_PATH__:/a\n")
        let resolver = LoginPathResolver(runner: runner, shell: nil, home: nil)
        _ = await resolver.path()
        #expect(runner.calls.map(\.shell) == ["/bin/zsh"])
    }

    @Test func fallsBackWhenTheShellFails() async throws {
        let home = try TemporaryDirectory()
        try home.makeDirectory(".nvm/versions/node/v20/bin")
        try home.makeDirectory(".nvm/versions/node/v18/bin")
        try home.makeDirectory("Library/Application Support/Herd/config/nvm/versions/node/v22")
        let resolver = LoginPathResolver(runner: StubShell(output: nil), shell: "/bin/zsh", home: home.path)
        let base = home.path
        let expected = [
            "/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin",
            base + "/.bun/bin", base + "/.local/bin", base + "/.cargo/bin",
            base + "/.nvm/versions/node/v18/bin", base + "/.nvm/versions/node/v20/bin",
            base + "/Library/Application Support/Herd/config/nvm/versions/node/v22/bin",
        ].joined(separator: ":")
        #expect(await resolver.path() == expected)
    }

    @Test func fallbackWithoutHomeIsTheSystemDirectories() async {
        let resolver = LoginPathResolver(runner: StubShell(output: nil), shell: nil, home: nil)
        #expect(await resolver.path() == "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin")
    }

    @Test func blankMarkerFallsBack() async {
        let resolver = LoginPathResolver(runner: StubShell(output: "__ORBIT_PATH__:   \n"), shell: nil, home: nil)
        #expect(await resolver.path() == "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin")
    }

    @Test func applyAddsPathWhenMissing() async {
        let resolver = LoginPathResolver(runner: StubShell(output: "__ORBIT_PATH__:/a:/b\n"), shell: nil, home: nil)
        let result = await resolver.apply(to: EnvironmentList(["FOO=bar"]))
        #expect(result.entries == ["FOO=bar", "PATH=/a:/b"])
    }

    @Test func applyMergesExistingFirstAndLeavesTheInputUntouched() async {
        let resolver = LoginPathResolver(runner: StubShell(output: "__ORBIT_PATH__:/a:/b\n"), shell: nil, home: nil)
        let input = EnvironmentList(["PATH=/custom/bin:/b"])
        let result = await resolver.apply(to: input)
        #expect(result.entries == ["PATH=/custom/bin:/b:/a"])
        #expect(input.entries == ["PATH=/custom/bin:/b"])
    }
}
