extension Error {
    public func contains(_ error: RuntimeError) -> Bool {
        if let aggregate = self as? RuntimeErrors {
            return aggregate.contains(error)
        }
        return (self as? RuntimeError)?.matches(error) ?? false
    }
}
