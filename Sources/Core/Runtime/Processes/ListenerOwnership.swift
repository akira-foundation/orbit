public enum ListenerOwnership {
    public static func verify(
        supervisor: ProcessIdentity, endpoint: Endpoint, inspector: any ProcessInspector
    ) throws {
        try inspector.verifyIdentity(supervisor)
        let owners = try inspector.listenerOwners(endpoint)
        guard !owners.isEmpty else { throw RuntimeError.unownedEndpoint(nil) }
        for owner in owners {
            try inspector.verifyIdentity(owner)
            guard owner.pgid == supervisor.pgid else { throw RuntimeError.unownedEndpoint(nil) }
        }
        let confirmed = try inspector.listenerOwners(endpoint)
        guard IdentitySet.same(owners, confirmed) else {
            throw RuntimeError.unownedEndpoint("listener owner set changed during inspection")
        }
    }
}
