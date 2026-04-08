import Foundation
import Darwin

/// The Unix domain socket path used for frontend-backend communication.
/// Must match the -socket flag passed to gitctl-backend (default: /tmp/gitctl.sock).
let gitctlSocketPath = "/tmp/gitctl.sock"

/// A URLProtocol that routes HTTP requests over a Unix domain socket using
/// POSIX APIs (no Network.framework entitlements required).
///
/// Uses HTTP/1.0 so the response body is terminated by connection close —
/// no chunked transfer encoding to decode.
class UnixSocketProtocol: URLProtocol {
    private var ioTask: Task<Void, Never>?

    override class func canInit(with request: URLRequest) -> Bool {
        return request.url?.scheme == "http"
    }

    override class func canonicalRequest(for request: URLRequest) -> URLRequest {
        return request
    }

    override func startLoading() {
        let req = self.request
        ioTask = Task.detached(priority: .userInitiated) { [weak self] in
            guard let self else { return }
            do {
                try self.performRequest(req)
            } catch {
                self.client?.urlProtocol(self, didFailWithError: error)
            }
        }
    }

    private func performRequest(_ req: URLRequest) throws {
        let sock = socket(AF_UNIX, SOCK_STREAM, 0)
        guard sock >= 0 else {
            throw URLError(.cannotConnectToHost)
        }
        defer { close(sock) }

        var addr = sockaddr_un()
        addr.sun_family = sa_family_t(AF_UNIX)
        let pathBytes = Array(gitctlSocketPath.utf8)
        guard pathBytes.count < MemoryLayout.size(ofValue: addr.sun_path) else {
            throw URLError(.cannotConnectToHost)
        }
        withUnsafeMutableBytes(of: &addr.sun_path) { ptr in
            for (i, byte) in pathBytes.enumerated() { ptr[i] = byte }
        }

        let connectResult = withUnsafePointer(to: &addr) {
            $0.withMemoryRebound(to: sockaddr.self, capacity: 1) {
                Darwin.connect(sock, $0, socklen_t(MemoryLayout<sockaddr_un>.size))
            }
        }
        guard connectResult == 0 else {
            throw URLError(.cannotConnectToHost)
        }

        let payload = buildPayload(req)
        try sendAll(sock: sock, data: payload)
        shutdown(sock, SHUT_WR)

        let responseData = try readAll(sock: sock)
        try deliverResponse(responseData, for: req)
    }

    private func buildPayload(_ req: URLRequest) -> Data {
        guard let url = req.url else { return Data() }
        let method = req.httpMethod ?? "GET"
        var path = url.path.isEmpty ? "/" : url.path
        if let query = url.query { path += "?\(query)" }

        var header = "\(method) \(path) HTTP/1.0\r\nHost: localhost\r\n"
        req.allHTTPHeaderFields?.forEach { key, value in
            if key.lowercased() != "host" { header += "\(key): \(value)\r\n" }
        }

        var payload = Data(header.utf8)
        if let body = req.httpBody {
            payload += Data("Content-Length: \(body.count)\r\n\r\n".utf8)
            payload.append(body)
        } else {
            payload += Data("\r\n".utf8)
        }
        return payload
    }

    private func sendAll(sock: Int32, data: Data) throws {
        var offset = 0
        try data.withUnsafeBytes { (ptr: UnsafeRawBufferPointer) throws in
            guard let base = ptr.baseAddress else { return }
            while offset < data.count {
                let n = Darwin.send(sock, base.advanced(by: offset), data.count - offset, 0)
                if n <= 0 { throw URLError(.networkConnectionLost) }
                offset += n
            }
        }
    }

    private func readAll(sock: Int32) throws -> Data {
        var result = Data()
        var buf = [UInt8](repeating: 0, count: 4096)
        while true {
            let n = Darwin.recv(sock, &buf, buf.count, 0)
            if n < 0 { throw URLError(.networkConnectionLost) }
            if n == 0 { break }
            result.append(contentsOf: buf[..<n])
        }
        return result
    }

    private func deliverResponse(_ data: Data, for req: URLRequest) throws {
        let separator = Data("\r\n\r\n".utf8)
        guard let headerEnd = data.range(of: separator) else {
            throw URLError(.cannotParseResponse)
        }
        guard let headerString = String(data: data[..<headerEnd.lowerBound], encoding: .utf8) else {
            throw URLError(.cannotParseResponse)
        }

        let lines = headerString.components(separatedBy: "\r\n")
        let parts = lines.first?.split(separator: " ", maxSplits: 2) ?? []
        guard parts.count >= 2, let statusCode = Int(parts[1]) else {
            throw URLError(.cannotParseResponse)
        }

        var headers: [String: String] = [:]
        for line in lines.dropFirst() where !line.isEmpty {
            if let colon = line.firstIndex(of: ":") {
                let key = String(line[..<colon]).trimmingCharacters(in: .whitespaces)
                let val = String(line[line.index(after: colon)...]).trimmingCharacters(in: .whitespaces)
                headers[key] = val
            }
        }

        guard let response = HTTPURLResponse(
            url: req.url!,
            statusCode: statusCode,
            httpVersion: "HTTP/1.0",
            headerFields: headers
        ) else {
            throw URLError(.cannotParseResponse)
        }

        let body = Data(data[headerEnd.upperBound...])
        client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: body)
        client?.urlProtocolDidFinishLoading(self)
    }

    override func stopLoading() {
        ioTask?.cancel()
    }
}
