use agentsmesh_types::ServiceError;

pub fn wire<E: Into<ServiceError>>(e: E) -> String {
    e.into().to_wire()
}
