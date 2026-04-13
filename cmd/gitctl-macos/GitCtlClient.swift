import Foundation

/// HTTP client that talks to the gitctl backend over a Unix domain socket.
class GitCtlClient {
    let baseURL: URL
    let session: URLSession

    init(baseURL: URL = URL(string: "http://localhost")!) {
        self.baseURL = baseURL
        let config = URLSessionConfiguration.ephemeral
        config.protocolClasses = [UnixSocketProtocol.self]
        self.session = URLSession(configuration: config)
    }

    /// Creates a URLRequest with the Accept: text/html header so the backend
    /// returns markdown body fields pre-rendered as HTML.
    private func htmlRequest(url: URL) -> URLRequest {
        var request = URLRequest(url: url)
        request.setValue("text/html", forHTTPHeaderField: "Accept")
        return request
    }

    func listRepos(username: String) async throws -> [GitRepo] {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)!
        components.path = "/apis/gitctl.justinsb.com/v1alpha1/gitrepos"
        components.queryItems = [URLQueryItem(name: "username", value: username)]

        let (data, response) = try await session.data(from: components.url!)

        guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
            throw GitCtlError.badResponse
        }

        let repoList = try JSONDecoder().decode(GitRepoList.self, from: data)
        return repoList.items
    }

    func listPullRequests(username: String, scope: String) async throws -> [PullRequest] {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)!
        components.path = "/apis/gitctl.justinsb.com/v1alpha1/pullrequests"
        components.queryItems = [
            URLQueryItem(name: "username", value: username),
            URLQueryItem(name: "scope", value: scope),
        ]

        let (data, response) = try await session.data(for: htmlRequest(url: components.url!))

        guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
            throw GitCtlError.badResponse
        }

        let prList = try JSONDecoder().decode(PullRequestList.self, from: data)
        return prList.items
    }

    func listIssues(username: String, scope: String) async throws -> [Issue] {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)!
        components.path = "/apis/gitctl.justinsb.com/v1alpha1/issues"
        components.queryItems = [
            URLQueryItem(name: "username", value: username),
            URLQueryItem(name: "scope", value: scope),
        ]

        let (data, response) = try await session.data(for: htmlRequest(url: components.url!))

        guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
            throw GitCtlError.badResponse
        }

        let issueList = try JSONDecoder().decode(IssueList.self, from: data)
        return issueList.items
    }

    // MARK: - Views

    func listViews() async throws -> [View] {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)!
        components.path = "/apis/gitctl.justinsb.com/v1alpha1/views"

        let (data, response) = try await session.data(from: components.url!)

        guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
            throw GitCtlError.badResponse
        }

        let viewList = try JSONDecoder().decode(ViewList.self, from: data)
        return viewList.items
    }

    func createView(view: View) async throws -> View {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)!
        components.path = "/apis/gitctl.justinsb.com/v1alpha1/views"

        var request = URLRequest(url: components.url!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(view)

        let (data, response) = try await session.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 201 else {
            let statusCode = (response as? HTTPURLResponse)?.statusCode ?? 0
            let body = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            print("createView error: status=\(statusCode), response body=\(body)")
            throw GitCtlError.httpError(statusCode, body)
        }

        return try JSONDecoder().decode(View.self, from: data)
    }

    func updateView(view: View) async throws -> View {
        let name = view.metadata?.name ?? ""
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)!
        components.path = "/apis/gitctl.justinsb.com/v1alpha1/views/\(name)"

        var request = URLRequest(url: components.url!)
        request.httpMethod = "PUT"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(view)

        let (data, response) = try await session.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
            throw GitCtlError.badResponse
        }

        return try JSONDecoder().decode(View.self, from: data)
    }

    func deleteView(name: String) async throws {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)!
        components.path = "/apis/gitctl.justinsb.com/v1alpha1/views/\(name)"

        var request = URLRequest(url: components.url!)
        request.httpMethod = "DELETE"

        let (_, response) = try await session.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 204 else {
            throw GitCtlError.badResponse
        }
    }

    /// Parses a GitHub pulls/issues URL into a search query and display name.
    /// Returns nil if the URL is not a supported GitHub URL.
    func parseGitHubURL(urlString: String) async throws -> ParsedGitHubURL? {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)!
        components.path = "/apis/gitctl.justinsb.com/v1alpha1/parseurl"
        components.queryItems = [URLQueryItem(name: "url", value: urlString)]

        let (data, response) = try await session.data(from: components.url!)

        guard let httpResponse = response as? HTTPURLResponse else {
            return nil
        }
        if httpResponse.statusCode == 422 {
            return nil
        }
        guard httpResponse.statusCode == 200 else {
            return nil
        }

        return try JSONDecoder().decode(ParsedGitHubURL.self, from: data)
    }

    func executeView(name: String) async throws -> ViewResults {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)!
        components.path = "/apis/gitctl.justinsb.com/v1alpha1/views/\(name)/results"

        let (data, response) = try await URLSession.shared.data(for: htmlRequest(url: components.url!))

        guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
            throw GitCtlError.badResponse
        }

        return try JSONDecoder().decode(ViewResults.self, from: data)
    }

}

enum GitCtlError: LocalizedError {
    case badResponse
    case httpError(Int, String)  // statusCode, responseBody

    var errorDescription: String? {
        switch self {
        case .badResponse:
            return "Bad response from backend"
        case .httpError(let code, let body):
            let msg = body.isEmpty ? "HTTP \(code)" : "HTTP \(code): \(body)"
            return "Backend error (\(msg))"
        }
    }
}
