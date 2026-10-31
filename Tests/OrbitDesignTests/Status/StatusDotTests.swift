import SwiftUI
import Testing

@testable import OrbitDesign

@Suite struct StatusDotTests {
    @Test(arguments: [
        (StatusDot.Status.running, Color.green),
        (.idle, .yellow),
        (.stopped, .secondary),
        (.error, .red),
    ])
    func statusMapsToSystemColor(status: StatusDot.Status, expected: Color) {
        #expect(status.color == expected)
    }

    @Test func everyStatusHasADistinctColor() {
        let colors = StatusDot.Status.allCases.map(\.color)
        #expect(Set(colors.map(String.init(describing:))).count == colors.count)
    }
}
