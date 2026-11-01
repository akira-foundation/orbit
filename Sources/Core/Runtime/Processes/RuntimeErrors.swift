public struct RuntimeErrors: Error, CustomStringConvertible {
    public let errors: [any Error]

    public static func join(_ errors: [(any Error)?]) -> (any Error)? {
        let flattened = errors.compactMap { $0 }.flatMap { error -> [any Error] in
            (error as? RuntimeErrors)?.errors ?? [error]
        }
        switch flattened.count {
        case 0: return nil
        case 1: return flattened[0]
        default: return RuntimeErrors(errors: flattened)
        }
    }

    public func contains(_ error: RuntimeError) -> Bool {
        errors.contains { $0.contains(error) }
    }

    public var description: String {
        errors.map { "\($0)" }.joined(separator: "\n")
    }
}
