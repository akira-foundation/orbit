import Darwin

extension ExitStatus {
    func failure(_ subject: String) -> RuntimeError? {
        switch self {
        case .exited(0): nil
        case .exited(let code): .message("\(subject) exit status \(code)")
        case .signaled(let signal): .message("\(subject) terminated by signal \(signal)")
        case .unknown(let code): .message("\(subject) wait: \(String(cString: strerror(code)))")
        }
    }
}
