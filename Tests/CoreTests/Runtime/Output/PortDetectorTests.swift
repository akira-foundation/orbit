import Testing

@testable import Core

struct PortDetectorTests {
    @Test(arguments: [
        "Port 5173 is in use, trying another one...",
        "Port 5174 is in use, trying another one...",
        "error: listen EADDRINUSE: address already in use :::3000",
        "⚠️  Port 3000 is already in use",
    ])
    func extractPortIgnoresConflictLines(line: String) {
        #expect(PortDetector.port(in: line) == nil)
    }

    @Test func extractPortHandlesANSIColoredOutput() {
        let line =
            "  \u{1B}[32m➜\u{1B}[39m  \u{1B}[1mLocal\u{1B}[22m:   \u{1B}[36mhttp://localhost:\u{1B}[1m5177\u{1B}[22m/\u{1B}[39m"
        #expect(PortDetector.port(in: line) == DetectedPort(port: 5177, strong: true))
    }

    @Test func extractPortStillDetectsRealPorts() {
        #expect(PortDetector.port(in: "Local: http://localhost:5176/") == DetectedPort(port: 5176, strong: true))
        #expect(
            PortDetector.port(in: "ready - started server on 0.0.0.0:3000") == DetectedPort(port: 3000, strong: true))
        #expect(
            PortDetector.port(in: "Server listening at port: 4321") == DetectedPort(port: 4321, strong: false))
        #expect(PortDetector.port(in: "[::1]:8080") == DetectedPort(port: 8080, strong: true))
        #expect(PortDetector.port(in: "localhost:99999") == nil)
    }

    @Test func extractPortFallsBackToThePortFlagWhenTheHostPortIsOutOfRange() {
        #expect(PortDetector.port(in: "localhost:99999 port=4000") == DetectedPort(port: 4000, strong: false))
    }

    @Test func extractPortIgnoresPlainOutput() {
        #expect(PortDetector.port(in: "compiling...") == nil)
        #expect(PortDetector.port(in: "") == nil)
    }

    @Test func httpEndpointLowercasesAndUnbracketsIPv6() {
        #expect(
            PortDetector.httpEndpoint(in: "Listening on HTTP://LOCALHOST:5173/")
                == HTTPEndpoint(scheme: "http", host: "localhost", port: 5173))
        #expect(
            PortDetector.httpEndpoint(in: "at https://[::1]:8443")
                == HTTPEndpoint(scheme: "https", host: "::1", port: 8443))
        #expect(PortDetector.httpEndpoint(in: "http://localhost:0") == nil)
        #expect(PortDetector.httpEndpoint(in: "http://localhost:00") == nil)
        #expect(PortDetector.httpEndpoint(in: "http://localhost:70000") == nil)
        #expect(PortDetector.httpEndpoint(in: "no endpoint here") == nil)
    }

    @Test func httpEndpointStripsANSI() {
        let line = "\u{1B}[36mhttp://127.0.0.1:\u{1B}[1m3000\u{1B}[22m\u{1B}[39m"
        #expect(PortDetector.httpEndpoint(in: line) == HTTPEndpoint(scheme: "http", host: "127.0.0.1", port: 3000))
    }
}
