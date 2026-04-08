import Foundation
import Network

/// The Unix domain socket path used for frontend-backend communication.
/// Located in Application Support to follow macOS conventions.
let gitctlSocketPath: String = {
    let appSupport = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first!
    let dir = appSupport.appendingPathComponent("GitCtl")
    try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
    return dir.appendingPathComponent("gitctl.sock").path
}()

/// A URLProtocol that routes HTTP requests over a Unix domain socket.
///
/// Uses HTTP/1.0 to avoid chunked transfer encoding, keeping the response
/// parsing simple: headers end at the first blank line; body ends at EOF
/// (connection close).
class UnixSocketProtocol: URLProtocol {
    private var nwConnection: NWConnection?
    private var responseBuffer = Data()

    override class func canInit(with request: URLRequest) -> Bool {
        return request.url?.scheme == "http"
    }

    override class func canonicalRequest(for request: URLRequest) -> URLRequest {
        return request
    }

    override func startLoading() {
        let endpoint = NWEndpoint.unix(path: gitctlSocketPath)
        let conn = NWConnection(to: endpoint, using: .tcp)
        nwConnection = conn

        conn.stateUpdateHandler = { [weak self] state in
            guard let self else { return }
            switch state {
            case .ready:
                self.sendRequest()
            case .failed(let error):
                self.client?.urlProtocol(self, didFailWithError: error)
            default:
                break
            }
        }
        conn.start(queue: .global(qos: .userInitiated))
    }

    private func sendRequest() {
        guard let url = request.url else {
            client?.urlProtocol(self, didFailWithError: URLError(.badURL))
            return
        }

        let method = request.httpMethod ?? "GET"
        var path = url.path.isEmpty ? "/" : url.path
        if let query = url.query {
            path += "?\(query)"
        }

        // Use HTTP/1.0 so the server sends a plain response body terminated
        // by connection close — no chunked encoding to decode.
        var header = "\(method) \(path) HTTP/1.0\r\n"
        header += "Host: localhost\r\n"

        request.allHTTPHeaderFields?.forEach { key, value in
            let lower = key.lowercased()
            if lower != "host" {
                header += "\(key): \(value)\r\n"
            }
        }

        var payload = Data(header.utf8)
        if let body = request.httpBody {
            payload += "Content-Length: \(body.count)\r\n\r\n".data(using: .utf8)!
            payload.append(body)
        } else {
            payload += "\r\n".data(using: .utf8)!
        }

        nwConnection?.send(content: payload, completion: .contentProcessed { [weak self] error in
            guard let self else { return }
            if let error {
                self.client?.urlProtocol(self, didFailWithError: error)
            } else {
                self.readData()
            }
        })
    }

    private func readData() {
        nwConnection?.receive(minimumIncompleteLength: 1, maximumLength: 65536) { [weak self] data, _, isComplete, error in
            guard let self else { return }
            if let data, !data.isEmpty {
                self.responseBuffer.append(data)
            }
            if isComplete {
                self.deliverResponse()
            } else if let error {
                self.client?.urlProtocol(self, didFailWithError: error)
            } else {
                self.readData()
            }
        }
    }

    private func deliverResponse() {
        let separator = Data("\r\n\r\n".utf8)
        guard let headerEnd = responseBuffer.range(of: separator) else {
            client?.urlProtocol(self, didFailWithError: URLError(.cannotParseResponse))
            return
        }

        guard let headerString = String(data: responseBuffer[..<headerEnd.lowerBound], encoding: .utf8) else {
            client?.urlProtocol(self, didFailWithError: URLError(.cannotParseResponse))
            return
        }

        let lines = headerString.components(separatedBy: "\r\n")
        guard let statusLine = lines.first else {
            client?.urlProtocol(self, didFailWithError: URLError(.cannotParseResponse))
            return
        }

        // Parse "HTTP/1.0 200 OK"
        let parts = statusLine.split(separator: " ", maxSplits: 2)
        guard parts.count >= 2, let statusCode = Int(parts[1]) else {
            client?.urlProtocol(self, didFailWithError: URLError(.cannotParseResponse))
            return
        }

        var headers: [String: String] = [:]
        for line in lines.dropFirst() where !line.isEmpty {
            if let colon = line.firstIndex(of: ":") {
                let key = String(line[..<colon]).trimmingCharacters(in: .whitespaces)
                let value = String(line[line.index(after: colon)...]).trimmingCharacters(in: .whitespaces)
                headers[key] = value
            }
        }

        guard let response = HTTPURLResponse(
            url: request.url!,
            statusCode: statusCode,
            httpVersion: "HTTP/1.0",
            headerFields: headers
        ) else {
            client?.urlProtocol(self, didFailWithError: URLError(.cannotParseResponse))
            return
        }

        let body = Data(responseBuffer[headerEnd.upperBound...])
        client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: body)
        client?.urlProtocolDidFinishLoading(self)
    }

    override func stopLoading() {
        nwConnection?.cancel()
        nwConnection = nil
    }
}
