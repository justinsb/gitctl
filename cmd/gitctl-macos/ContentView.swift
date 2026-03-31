import SwiftUI

enum SidebarItem: String, CaseIterable, Identifiable {
    case feed = "Feed"
    case assigned = "Assigned"
    case repos = "Repositories"

    var id: String { rawValue }

    var systemImage: String {
        switch self {
        case .feed: return "arrow.up.circle"
        case .assigned: return "person.circle"
        case .repos: return "folder"
        }
    }
}

// MARK: - Selectable item for the detail column

enum SelectableItem: Identifiable, Hashable {
    case pullRequest(PullRequest)
    case issue(Issue)

    var id: String {
        switch self {
        case .pullRequest(let pr): return "pr-\(pr.id)"
        case .issue(let issue): return "issue-\(issue.id)"
        }
    }

    func hash(into hasher: inout Hasher) {
        hasher.combine(id)
    }

    static func == (lhs: SelectableItem, rhs: SelectableItem) -> Bool {
        lhs.id == rhs.id
    }
}

struct ContentView: View {
    @State private var sidebarSelection: SidebarItem? = .feed
    @State private var selectedItem: SelectableItem?

    var body: some View {
        if let selected = selectedItem {
            // Full-screen detail view
            Group {
                switch selected {
                case .pullRequest(let pr):
                    PRDetailView(pr: pr)
                case .issue(let issue):
                    IssueDetailView(issue: issue)
                }
            }
            .toolbar {
                ToolbarItem(placement: .navigation) {
                    Button(action: { selectedItem = nil }) {
                        Label("Back", systemImage: "chevron.left")
                    }
                    .keyboardShortcut(.escape, modifiers: [])
                }
            }
        } else {
            // List view with sidebar
            NavigationSplitView {
                List(selection: $sidebarSelection) {
                    ForEach(SidebarItem.allCases) { item in
                        NavigationLink(value: item) {
                            Label(item.rawValue, systemImage: item.systemImage)
                        }
                    }
                }
                .navigationTitle("gitctl")
            } detail: {
                switch sidebarSelection {
                case .feed:
                    FeedListView(selectedItem: $selectedItem)
                case .assigned:
                    AssignedListView(selectedItem: $selectedItem)
                case .repos:
                    ReposView()
                case nil:
                    Text("Select a section")
                        .foregroundStyle(.secondary)
                }
            }
        }
    }
}

// MARK: - Feed List View (Outbound PRs)

struct FeedListView: View {
    @Binding var selectedItem: SelectableItem?
    @State private var prs: [PullRequest] = []
    @State private var isLoading = true
    @State private var errorMessage: String?
    @State private var searchText = ""

    private let client = GitCtlClient()
    private let username = "justinsb"

    var filteredPRs: [PullRequest] {
        if searchText.isEmpty { return prs }
        return prs.filter { pr in
            let title = pr.spec?.title ?? ""
            let repo = pr.status?.repo ?? ""
            return title.localizedCaseInsensitiveContains(searchText)
                || repo.localizedCaseInsensitiveContains(searchText)
        }
    }

    var body: some View {
        Group {
            if isLoading {
                ProgressView("Loading feed...")
            } else if let error = errorMessage {
                ErrorView(message: error) { Task { await load() } }
            } else {
                List(selection: $selectedItem) {
                    ForEach(filteredPRs) { pr in
                        PRRow(pr: pr)
                            .tag(SelectableItem.pullRequest(pr))
                    }
                }
            }
        }
        .navigationTitle("Feed")
        .searchable(text: $searchText, prompt: "Filter PRs")
        .toolbar {
            ToolbarItem(placement: .automatic) {
                Button(action: { Task { await load() } }) {
                    Image(systemName: "arrow.clockwise")
                }
                .help("Refresh")
            }
        }
        .task { await load() }
    }

    func load() async {
        isLoading = true
        errorMessage = nil
        do {
            prs = try await client.listPullRequests(username: username, scope: "outbound")
            isLoading = false
        } catch {
            errorMessage = "Failed to load feed: \(error.localizedDescription)\n\nMake sure the backend is running:\n  go run cmd/gitctl-backend/main.go"
            isLoading = false
        }
    }
}

// MARK: - Assigned List View (PRs + Issues assigned to me)

struct AssignedListView: View {
    @Binding var selectedItem: SelectableItem?
    @State private var prs: [PullRequest] = []
    @State private var issues: [Issue] = []
    @State private var isLoading = true
    @State private var errorMessage: String?
    @State private var searchText = ""

    private let client = GitCtlClient()
    private let username = "justinsb"

    var filteredPRs: [PullRequest] {
        if searchText.isEmpty { return prs }
        return prs.filter { pr in
            let title = pr.spec?.title ?? ""
            let repo = pr.status?.repo ?? ""
            return title.localizedCaseInsensitiveContains(searchText)
                || repo.localizedCaseInsensitiveContains(searchText)
        }
    }

    var filteredIssues: [Issue] {
        if searchText.isEmpty { return issues }
        return issues.filter { issue in
            let title = issue.spec?.title ?? ""
            let repo = issue.status?.repo ?? ""
            return title.localizedCaseInsensitiveContains(searchText)
                || repo.localizedCaseInsensitiveContains(searchText)
        }
    }

    var body: some View {
        Group {
            if isLoading {
                ProgressView("Loading assigned items...")
            } else if let error = errorMessage {
                ErrorView(message: error) { Task { await load() } }
            } else {
                List(selection: $selectedItem) {
                    if !filteredPRs.isEmpty {
                        Section("Pull Requests") {
                            ForEach(filteredPRs) { pr in
                                PRRow(pr: pr)
                                    .tag(SelectableItem.pullRequest(pr))
                            }
                        }
                    }
                    if !filteredIssues.isEmpty {
                        Section("Issues") {
                            ForEach(filteredIssues) { issue in
                                IssueRow(issue: issue)
                                    .tag(SelectableItem.issue(issue))
                            }
                        }
                    }
                    if filteredPRs.isEmpty && filteredIssues.isEmpty {
                        Text("No items assigned to you")
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
        .navigationTitle("Assigned")
        .searchable(text: $searchText, prompt: "Filter items")
        .toolbar {
            ToolbarItem(placement: .automatic) {
                Button(action: { Task { await load() } }) {
                    Image(systemName: "arrow.clockwise")
                }
                .help("Refresh")
            }
        }
        .task { await load() }
    }

    func load() async {
        isLoading = true
        errorMessage = nil
        do {
            async let fetchPRs = client.listPullRequests(username: username, scope: "assigned")
            async let fetchIssues = client.listIssues(username: username, scope: "assigned")
            prs = try await fetchPRs
            issues = try await fetchIssues
            isLoading = false
        } catch {
            errorMessage = "Failed to load assigned items: \(error.localizedDescription)\n\nMake sure the backend is running:\n  go run cmd/gitctl-backend/main.go"
            isLoading = false
        }
    }
}

// MARK: - Repos View

struct ReposView: View {
    @State private var repos: [GitRepo] = []
    @State private var isLoading = true
    @State private var errorMessage: String?
    @State private var searchText = ""

    private let client = GitCtlClient()
    private let username = "justinsb"

    var filteredRepos: [GitRepo] {
        if searchText.isEmpty { return repos }
        return repos.filter { repo in
            let name = repo.metadata?.name ?? ""
            let desc = repo.spec?.description ?? ""
            return name.localizedCaseInsensitiveContains(searchText)
                || desc.localizedCaseInsensitiveContains(searchText)
        }
    }

    var body: some View {
        Group {
            if isLoading {
                ProgressView("Loading repositories...")
            } else if let error = errorMessage {
                ErrorView(message: error) { Task { await load() } }
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
                Button(action: { Task { await load() } }) {
                    Image(systemName: "arrow.clockwise")
                }
                .help("Refresh")
            }
        }
        .task { await load() }
    }

    func load() async {
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

// MARK: - Detail Views

struct PRDetailView: View {
    let pr: PullRequest

    private var detailURL: URL {
        let repo = pr.status?.repo ?? ""
        let number = pr.status?.number ?? 0
        // repo is "owner/repo", split for URL path
        return URL(string: "http://localhost:8484/ui/repos/\(repo)/pulls/\(number)")!
    }

    var body: some View {
        DetailWebView(url: detailURL)
            .navigationTitle("\(pr.status?.repo ?? "")#\(pr.status?.number ?? 0)")
    }
}

struct IssueDetailView: View {
    let issue: Issue

    private var detailURL: URL {
        let repo = issue.status?.repo ?? ""
        let number = issue.status?.number ?? 0
        return URL(string: "http://localhost:8484/ui/repos/\(repo)/issues/\(number)")!
    }

    var body: some View {
        DetailWebView(url: detailURL)
            .navigationTitle("\(issue.status?.repo ?? "")#\(issue.status?.number ?? 0)")
    }
}

// MARK: - Row Views

struct PRRow: View {
    let pr: PullRequest

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text("\(pr.status?.repo ?? "")#\(pr.status?.number ?? 0)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                Text(pr.spec?.title ?? "untitled")
                    .font(.headline)
                    .lineLimit(1)
                Spacer()
                if pr.status?.draft == true {
                    Text("Draft")
                        .font(.caption2)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(.quaternary)
                        .clipShape(Capsule())
                }
                if pr.status?.merged == true {
                    Text("Merged")
                        .font(.caption2)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(.purple.opacity(0.2))
                        .clipShape(Capsule())
                }
            }

            HStack(spacing: 12) {
                Label(pr.status?.author ?? "", systemImage: "person")
                if let updated = pr.status?.updatedAt, updated.count >= 10 {
                    Label(String(updated.prefix(10)), systemImage: "clock")
                }
            }
            .font(.caption)
            .foregroundStyle(.secondary)

            if let labels = pr.status?.labels, !labels.isEmpty {
                HStack(spacing: 4) {
                    ForEach(labels, id: \.self) { label in
                        Text(label)
                            .font(.caption2)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 1)
                            .background(.blue.opacity(0.15))
                            .clipShape(Capsule())
                    }
                }
            }
        }
        .padding(.vertical, 4)
    }
}

struct IssueRow: View {
    let issue: Issue

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text("\(issue.status?.repo ?? "")#\(issue.status?.number ?? 0)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                Text(issue.spec?.title ?? "untitled")
                    .font(.headline)
                    .lineLimit(1)
                Spacer()
            }

            HStack(spacing: 12) {
                Label(issue.status?.author ?? "", systemImage: "person")
                if let updated = issue.status?.updatedAt, updated.count >= 10 {
                    Label(String(updated.prefix(10)), systemImage: "clock")
                }
            }
            .font(.caption)
            .foregroundStyle(.secondary)

            if let labels = issue.status?.labels, !labels.isEmpty {
                HStack(spacing: 4) {
                    ForEach(labels, id: \.self) { label in
                        Text(label)
                            .font(.caption2)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 1)
                            .background(.blue.opacity(0.15))
                            .clipShape(Capsule())
                    }
                }
            }
        }
        .padding(.vertical, 4)
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

// MARK: - Shared Components

struct ErrorView: View {
    let message: String
    let retry: () -> Void

    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: "exclamationmark.triangle")
                .font(.largeTitle)
                .foregroundStyle(.secondary)
            Text(message)
                .foregroundStyle(.secondary)
            Button("Retry", action: retry)
        }
        .padding()
    }
}
