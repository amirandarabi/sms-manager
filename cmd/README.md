# cmd

One `main` package per deployable binary (gateway-api, dispatcher, dlr-service, callback-dispatcher, scheduler, outbox-relay, otp-service, content-guard, admin-api, sdk-generator — see ARCHITECTURE.md §18.1). Each subdirectory wires together `pkg/*` and `internal/*` code into a runnable service; no business logic lives here.
