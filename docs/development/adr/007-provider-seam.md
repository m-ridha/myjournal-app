# 007: Provider seam

A provider interface is used to hide sign-in detail and the storage backend behind a small abstraction. The generic code deals with a connection, not Google-specific objects, which keeps credentials, Drive folder logic, and routing isolated from the domain model.
