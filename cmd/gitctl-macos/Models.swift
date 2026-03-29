import Foundation

// Kubernetes-style API types matching the Go backend's JSON wire format.

struct ObjectMeta: Codable, Hashable {
    var name: String?
    var namespace: String?
}

struct ListMeta: Codable {}

struct GitRepoSpec: Codable {
    var description: String?
    var `private`: Bool?
}

struct GitRepoStatus: Codable {
    var fullName: String?
    var htmlUrl: String?
    var fork: Bool?
    var stargazersCount: Int?
    var forksCount: Int?
    var openIssuesCount: Int?
    var language: String?
    var createdAt: String?
    var updatedAt: String?
    var pushedAt: String?
}

struct GitRepo: Codable, Identifiable {
    var apiVersion: String?
    var kind: String?
    var metadata: ObjectMeta?
    var spec: GitRepoSpec?
    var status: GitRepoStatus?

    var id: String { metadata?.name ?? UUID().uuidString }
}

struct GitRepoList: Codable {
    var apiVersion: String?
    var kind: String?
    var metadata: ListMeta?
    var items: [GitRepo]
}

// MARK: - PullRequest

struct PullRequestSpec: Codable, Hashable {
    var title: String?
    var body: String?
}

struct PullRequestStatus: Codable, Hashable {
    var repo: String?
    var number: Int?
    var state: String?
    var author: String?
    var assignees: [String]?
    var htmlUrl: String?
    var draft: Bool?
    var merged: Bool?
    var labels: [String]?
    var createdAt: String?
    var updatedAt: String?
}

struct PullRequest: Codable, Identifiable, Hashable {
    var apiVersion: String?
    var kind: String?
    var metadata: ObjectMeta?
    var spec: PullRequestSpec?
    var status: PullRequestStatus?

    var id: String { metadata?.name ?? UUID().uuidString }
}

struct PullRequestList: Codable {
    var apiVersion: String?
    var kind: String?
    var metadata: ListMeta?
    var items: [PullRequest]
}

// MARK: - Issue

struct IssueSpec: Codable, Hashable {
    var title: String?
    var body: String?
}

struct IssueStatus: Codable, Hashable {
    var repo: String?
    var number: Int?
    var state: String?
    var author: String?
    var assignees: [String]?
    var htmlUrl: String?
    var labels: [String]?
    var createdAt: String?
    var updatedAt: String?
}

struct Issue: Codable, Identifiable, Hashable {
    var apiVersion: String?
    var kind: String?
    var metadata: ObjectMeta?
    var spec: IssueSpec?
    var status: IssueStatus?

    var id: String { metadata?.name ?? UUID().uuidString }
}

struct IssueList: Codable {
    var apiVersion: String?
    var kind: String?
    var metadata: ListMeta?
    var items: [Issue]
}

// MARK: - Comment

struct CommentSpec: Codable {
    var body: String?
}

struct CommentStatus: Codable {
    var author: String?
    var htmlUrl: String?
    var createdAt: String?
    var updatedAt: String?
}

struct Comment: Codable, Identifiable {
    var apiVersion: String?
    var kind: String?
    var metadata: ObjectMeta?
    var spec: CommentSpec?
    var status: CommentStatus?

    var id: String { metadata?.name ?? UUID().uuidString }
}

struct CommentList: Codable {
    var apiVersion: String?
    var kind: String?
    var metadata: ListMeta?
    var items: [Comment]
}
