import Darwin
import Foundation

enum ProcessFixture {
    static let runtimeFixtureURL: URL = {
        var info = Dl_info()
        guard dladdr(#dsohandle, &info) != 0, let image = info.dli_fname else {
            return URL(filePath: "/nonexistent/orbit-runtime-fixture")
        }
        var url = URL(filePath: String(cString: image)).resolvingSymlinksInPath()
        while url.pathExtension != "xctest" && url.path != "/" {
            url.deleteLastPathComponent()
        }
        return url.deletingLastPathComponent().appending(component: "orbit-runtime-fixture")
    }()

    static var runtimeFixturePath: String {
        runtimeFixtureURL.path(percentEncoded: false)
    }
}
