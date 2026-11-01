import Darwin
import Testing

enum ThreadSanitizer {
    static let isLoaded = dlsym(UnsafeMutableRawPointer(bitPattern: -2), "__tsan_init") != nil
    static let exitTestsUnsupported: Comment = "exit test children are relaunched without the sanitizer runtime"
}
