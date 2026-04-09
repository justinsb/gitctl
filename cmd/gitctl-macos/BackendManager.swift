import Foundation
import os

/// Manages the lifecycle of the gitctl-backend process bundled inside the app.
class BackendManager: ObservableObject {
    private var process: Process?
    private let logger = Logger(subsystem: "com.justinsb.gitctl", category: "BackendManager")

    /// Starts the backend binary from the app bundle, then waits for it to become ready.
    func start() {
        guard process == nil else { return }

        guard let executableURL = Bundle.main.executableURL else {
            logger.error("Cannot determine app bundle executable path")
            return
        }

        let backendURL = executableURL
            .deletingLastPathComponent()
            .appendingPathComponent("gitctl-backend")

        guard FileManager.default.fileExists(atPath: backendURL.path) else {
            logger.error("gitctl-backend not found at \(backendURL.path)")
            return
        }

        let proc = Process()
        proc.executableURL = backendURL
        proc.arguments = ["-socket=\(gitctlSocketPath)", "-tcp="]  // disable TCP, use socket only

        // Send backend logs to the system log.
        let pipe = Pipe()
        proc.standardOutput = pipe
        proc.standardError = pipe

        pipe.fileHandleForReading.readabilityHandler = { [logger] handle in
            let data = handle.availableData
            if !data.isEmpty, let line = String(data: data, encoding: .utf8) {
                logger.info("backend: \(line, privacy: .public)")
            }
        }

        proc.terminationHandler = { [weak self, logger] proc in
            logger.info("Backend exited with status \(proc.terminationStatus)")
            DispatchQueue.main.async {
                self?.process = nil
            }
        }

        do {
            try proc.run()
            process = proc
            logger.info("Started backend (pid \(proc.processIdentifier))")
        } catch {
            logger.error("Failed to start backend: \(error.localizedDescription)")
            return
        }

        // Wait for the backend to become ready by polling /readyz.
        Task {
            await waitForBackendReady()
        }
    }

    /// Polls /readyz until the backend returns 200 OK or a timeout is reached.
    /// Polls every 100ms for up to 10 seconds.
    private func waitForBackendReady() async {
        let config = URLSessionConfiguration.ephemeral
        config.protocolClasses = [UnixSocketProtocol.self]
        let session = URLSession(configuration: config)

        guard let readyzURL = URL(string: "http://localhost/readyz") else { return }

        let deadline = Date().addingTimeInterval(10)
        while Date() < deadline {
            do {
                let (_, response) = try await session.data(from: readyzURL)
                if let http = response as? HTTPURLResponse, http.statusCode == 200 {
                    logger.info("Backend is ready")
                    return
                }
            } catch {
                // Connection refused or other transient error — keep polling.
            }
            try? await Task.sleep(nanoseconds: 100_000_000) // 100ms
        }
        logger.warning("Backend did not become ready within 10 seconds")
    }

    /// Stops the backend process gracefully.
    func stop() {
        guard let proc = process, proc.isRunning else { return }
        proc.terminate() // sends SIGTERM
        logger.info("Sent SIGTERM to backend (pid \(proc.processIdentifier))")
        process = nil
    }
}
