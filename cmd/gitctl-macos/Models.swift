import Foundation

// Kubernetes-style API types matching the Go backend's JSON wire format.

struct ObjectMeta: Codable {
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
