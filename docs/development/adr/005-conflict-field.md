# 005: Conflict detection field

The exact Drive field used for conflict detection remains subject to the Phase 1 spike results, but the implementation will keep the field opaque to the core domain and compare it only as a version token. This follows the port design and leaves room for the verified provider-specific choice.
