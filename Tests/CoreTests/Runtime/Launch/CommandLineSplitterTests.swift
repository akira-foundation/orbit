import Testing

@testable import Core

struct CommandLineSplitterTests {
    @Test(arguments: [
        ("npm run dev", ["npm", "run", "dev"]),
        ("  vite   --host  ", ["vite", "--host"]),
        ("sh -c \"echo ready now\"", ["sh", "-c", "echo ready now"]),
        ("a \"\" b", ["a", "b"]),
        ("a\tb", ["a\tb"]),
        ("", []),
    ])
    func splitCases(line: String, expected: [String]) {
        #expect(CommandLineSplitter.split(line) == expected)
    }
}
