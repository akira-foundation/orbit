@testable import Core

enum SettlingIdentity {
    static func executable(of pid: Int32, inspector: DarwinProcessInspector) throws -> String? {
        do {
            return try inspector.identity(of: pid).executable
        } catch let error as RuntimeError where error.matches(.identityMismatch("")) {
            return nil
        }
    }
}
