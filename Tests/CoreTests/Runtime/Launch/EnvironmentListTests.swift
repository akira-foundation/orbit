import Testing

@testable import Core

struct EnvironmentListTests {
    @Test func removeDropsEveryEntryAndKeepsOrder() {
        var env = EnvironmentList(["HOME=/h", "CI=true", "PATH=/bin", "CI=1"])
        env.remove("CI")
        #expect(env.entries == ["HOME=/h", "PATH=/bin"])
    }

    @Test func removeIsANoopWhenAbsent() {
        var env = EnvironmentList(["HOME=/h", "PATH=/bin"])
        env.remove("CI")
        #expect(env.entries == ["HOME=/h", "PATH=/bin"])
    }

    @Test func addAbsentKeepsExistingKeys() {
        var env = EnvironmentList(["DATABASE_URL=keep"])
        env.addAbsent(["DATABASE_URL": "other", "DB_HOST": "127.0.0.1"])
        #expect(env.entries == ["DATABASE_URL=keep", "DB_HOST=127.0.0.1"])
    }

    @Test func addAbsentAppendsInKeyOrder() {
        var env = EnvironmentList([])
        env.addAbsent(["B": "2", "A": "1"])
        #expect(env.entries == ["A=1", "B=2"])
    }

    @Test func overrideReplacesInPlaceAndAppendsNew() {
        var env = EnvironmentList(["A=1", "B=2"])
        env.override(["A": "9", "C": "3"])
        #expect(env.entries == ["A=9", "B=2", "C=3"])
    }

    @Test func setReplacesInPlaceAndAppendsWhenMissing() {
        var env = EnvironmentList(["A=1", "B=2"])
        env.set("A", "5")
        env.set("Z", "7")
        #expect(env.entries == ["A=5", "B=2", "Z=7"])
    }

    @Test func setReplacesTheLastDuplicate() {
        var env = EnvironmentList(["A=1", "B=2", "A=3"])
        env.set("A", "9")
        #expect(env.entries == ["A=1", "B=2", "A=9"])
    }

    @Test func valueReturnsFirstMatchOrNil() {
        let env = EnvironmentList(["A=1", "A=2", "B=x=y"])
        #expect(env.value(of: "A") == "1")
        #expect(env.value(of: "B") == "x=y")
        #expect(env.value(of: "C") == nil)
    }

    @Test func processListsTheCurrentEnvironment() {
        #expect(EnvironmentList.process().value(of: "PATH") != nil)
    }
}
