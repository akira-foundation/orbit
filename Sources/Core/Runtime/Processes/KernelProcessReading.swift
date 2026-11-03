public protocol KernelProcessReading: Sendable {
    func snapshot(_ pid: Int32) throws -> KernelProcessSnapshot
    func group(_ pgid: Int32) throws -> [KernelProcessSnapshot]
    func all() throws -> [KernelProcessSnapshot]
    func arguments(_ pid: Int32) throws -> String
}
