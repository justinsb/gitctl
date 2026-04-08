import Foundation
import os

/// Manages the lifecycle of the gitctl-backend process bundled inside the app.
class BackendManager: ObservableObject {
    private var process: Process?
    private let logger = Logger(subsystem: "com.justinsb.gitctl", category: "BackendManager")

    /// Starts the backend binary from the app bundle.
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
        proc.arguments = ["-socket=\(gitctlSocketPath)", "-tcp="]

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
        }
    }

    /// Stops the backend process gracefully.
    func stop() {
        guard let proc = process, proc.isRunning else { return }
        proc.terminate() // sends SIGTERM
        logger.info("Sent SIGTERM to backend (pid \(proc.processIdentifier))")
        process = nil
    }
}
