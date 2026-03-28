import Foundation

/// HTTP client that talks to the gitctl backend over TCP.
class GitCtlClient {
    let baseURL: URL

    init(baseURL: URL = URL(string: "http://localhost:8484")!) {
        self.baseURL = baseURL
    }

    func listRepos(username: String) async throws -> [GitRepo] {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)!
        components.path = "/apis/gitctl.justinsb.com/v1alpha1/gitrepos"
        components.queryItems = [URLQueryItem(name: "username", value: username)]

        let (data, response) = try await URLSession.shared.data(from: components.url!)

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

        let (data, response) = try await URLSession.shared.data(from: components.url!)

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

        let (data, response) = try await URLSession.shared.data(from: components.url!)

        guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
            throw GitCtlError.badResponse
        }

        let issueList = try JSONDecoder().decode(IssueList.self, from: data)
        return issueList.items
    }
}

enum GitCtlError: LocalizedError {
    case badResponse

    var errorDescription: String? {
        switch self {
        case .badResponse:
            return "Bad response from backend"
        }
    }
}
