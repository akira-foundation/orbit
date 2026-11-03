public enum ListenerOwnership {
    public static func verify(
        supervisor: ProcessIdentity, endpoint: Endpoint, inspector: any ProcessInspector
    ) throws {
        try inspector.verifyIdentity(supervisor)
        let listing = try inspector.listenerListing(endpoint)
        guard listing.unsettled.isEmpty else {
            throw RuntimeError.unownedEndpoint("listener owner did not settle during inspection")
        }
        let owners = listing.identities
        guard !owners.isEmpty else { throw RuntimeError.unownedEndpoint(nil) }
        for owner in owners {
            try inspector.verifyIdentity(owner)
            guard owner.pgid == supervisor.pgid else { throw RuntimeError.unownedEndpoint(nil) }
        }
        let confirmed = try inspector.listenerListing(endpoint)
        guard confirmed.unsettled.isEmpty, IdentitySet.same(owners, confirmed.identities) else {
            throw RuntimeError.unownedEndpoint("listener owner set changed during inspection")
        }
    }
}
