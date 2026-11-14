enum SupervisorTrigger: Sendable {
    case childExited(ExitStatus)
    case stopRequested
}
