import Core
import Foundation

if let code = RuntimeSupervisor.runIfRequested(arguments: Array(CommandLine.arguments.dropFirst())) {
    exit(code)
}

_ = OrbitEnvironment.current(bundleIdentifier: Bundle.main.bundleIdentifier)
