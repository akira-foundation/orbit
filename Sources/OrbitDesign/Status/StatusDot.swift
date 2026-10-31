import SwiftUI

public struct StatusDot: View {
    public enum Status: Sendable, CaseIterable {
        case running
        case idle
        case stopped
        case error

        public var color: Color {
            switch self {
            case .running: .green
            case .idle: .yellow
            case .stopped: .secondary
            case .error: .red
            }
        }
    }

    private let status: Status

    public init(_ status: Status) {
        self.status = status
    }

    public var body: some View {
        Circle()
            .fill(status.color)
            .frame(width: 8, height: 8)
            .accessibilityHidden(true)
    }
}
