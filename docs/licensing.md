# Licensing and Runtime Policy

This modern codebase is designed to run without proprietary runtime licensing checks.

## Policy

- No end-user license prompt gates on startup.
- No dependency on closed-source SDKs or drivers for core service startup.
- No embedded third-party binaries required for baseline API operation.

## Allowed dependencies

- Go standard library
- OSS dependencies with permissive licenses (MIT/BSD/Apache-2.0)

## Migration note

Legacy package assets under `../mcs` include proprietary/distribution-specific artifacts. They are treated as migration input only and are not required to start `modern-mcs`.
