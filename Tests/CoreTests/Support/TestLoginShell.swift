import Foundation

struct TestLoginShell {
    private let directory: TemporaryDirectory

    let path: String

    init() throws {
        directory = try TemporaryDirectory()
        path = directory.path + "/login-shell"
        try directory.write("#!/bin/sh\nshift\nexec /bin/sh -c \"$1\"\n", to: "login-shell")
        try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: path)
    }
}
