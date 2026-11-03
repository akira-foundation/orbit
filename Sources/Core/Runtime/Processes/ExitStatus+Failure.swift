import Darwin

extension ExitStatus {
    func failure(_ subject: String? = nil) -> RuntimeError? {
        let prefix = subject.map { "\($0) " } ?? ""
        switch self {
        case .exited(0): return nil
        case .exited(let code): return .message("\(prefix)exit status \(code)")
        case .signaled(let signal): return .message("\(prefix)terminated by signal \(signal)")
        case .unknown(let code): return .message("\(prefix)wait: \(String(cString: strerror(code)))")
        }
    }
}
