import SwiftUI

struct ContentView: View {
    @State private var repos: [GitRepo] = []
    @State private var isLoading = true
    @State private var errorMessage: String?
    @State private var searchText = ""

    private let client = GitCtlClient()
    private let username = "justinsb"

    var filteredRepos: [GitRepo] {
        if searchText.isEmpty {
            return repos
        }
        return repos.filter { repo in
            let name = repo.metadata?.name ?? ""
            let desc = repo.spec?.description ?? ""
            return name.localizedCaseInsensitiveContains(searchText)
                || desc.localizedCaseInsensitiveContains(searchText)
        }
    }

    var body: some View {
        NavigationSplitView {
            Group {
                if isLoading {
                    ProgressView("Loading repositories...")
                } else if let error = errorMessage {
                    VStack(spacing: 12) {
                        Image(systemName: "exclamationmark.triangle")
                            .font(.largeTitle)
                            .foregroundStyle(.secondary)
                        Text(error)
                            .foregroundStyle(.secondary)
                        Button("Retry") { Task { await loadRepos() } }
                    }
                    .padding()
                } else {
                    List(filteredRepos) { repo in
                        RepoRow(repo: repo)
                    }
                }
            }
            .navigationTitle("Repositories")
            .searchable(text: $searchText, prompt: "Filter repos")
            .toolbar {
                ToolbarItem(placement: .automatic) {
                    Button(action: { Task { await loadRepos() } }) {
                        Image(systemName: "arrow.clockwise")
                    }
                    .help("Refresh")
                }
            }
        } detail: {
            Text("Select a repository")
                .foregroundStyle(.secondary)
        }
        .task { await loadRepos() }
    }

    func loadRepos() async {
        isLoading = true
        errorMessage = nil
        do {
            repos = try await client.listRepos(username: username)
            isLoading = false
        } catch {
            errorMessage = "Failed to load repos: \(error.localizedDescription)\n\nMake sure the backend is running:\n  go run cmd/gitctl-backend/main.go"
            isLoading = false
        }
    }
}

struct RepoRow: View {
    let repo: GitRepo

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text(repo.metadata?.name ?? "unknown")
                    .font(.headline)
                Spacer()
                if let lang = repo.status?.language, !lang.isEmpty {
                    Text(lang)
                        .font(.caption)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(.quaternary)
                        .clipShape(Capsule())
                }
            }

            if let desc = repo.spec?.description, !desc.isEmpty {
                Text(desc)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }

            HStack(spacing: 12) {
                Label("\(repo.status?.stargazersCount ?? 0)", systemImage: "star")
                Label("\(repo.status?.forksCount ?? 0)", systemImage: "tuningfork")
                Label("\(repo.status?.openIssuesCount ?? 0)", systemImage: "exclamationmark.circle")
            }
            .font(.caption)
            .foregroundStyle(.secondary)
        }
        .padding(.vertical, 4)
    }
}
