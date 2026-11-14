@testable import Core

struct FixedEndpoint: EndpointReserving {
    let endpoint: Endpoint

    func reserve() throws -> Endpoint {
        endpoint
    }
}
