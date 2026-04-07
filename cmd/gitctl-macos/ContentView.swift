import SwiftUI

// MARK: - Sidebar Selection

enum SidebarSelection: Identifiable, Hashable {
    case feed
    case assigned
    case repos
    case view(View)

    var id: String {
        switch self {
        case .feed: return "feed"
        case .assigned: return "assigned"
        case .repos: return "repos"
        case .view(let v): return "view-\(v.id)"
        }
    }

    var label: String {
        switch self {
        case .feed: return "Feed"
        case .assigned: return "Assigned"
        case .repos: return "Repositories"
        case .view(let v): return v.spec?.displayName ?? v.metadata?.name ?? "Untitled"
        }
    }

    var systemImage: String {
        switch self {
        case .feed: return "arrow.up.circle"
        case .assigned: return "person.circle"
        case .repos: return "folder"
        case .view: return "magnifyingglass"
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

struct ContentView: SwiftUI.View {
    @State private var sidebarSelection: SidebarSelection? = .feed
    @State private var selectedItem: SelectableItem?
    @State private var views: [View] = []
    @State private var showCreateView = false
    @State private var viewToEdit: View? = nil

    private let client = GitCtlClient()

    var body: some SwiftUI.View {
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
                    Section("Navigation") {
                        NavigationLink(value: SidebarSelection.feed) {
                            Label("Feed", systemImage: "arrow.up.circle")
                        }
                        NavigationLink(value: SidebarSelection.assigned) {
                            Label("Assigned", systemImage: "person.circle")
                        }
                        NavigationLink(value: SidebarSelection.repos) {
                            Label("Repositories", systemImage: "folder")
                        }
                    }
                    Section("Views") {
                        ForEach(views) { view in
                            NavigationLink(value: SidebarSelection.view(view)) {
                                Label(view.spec?.displayName ?? view.metadata?.name ?? "Untitled",
                                      systemImage: "magnifyingglass")
                            }
                            .contextMenu {
                                Button {
                                    viewToEdit = view
                                } label: {
                                    Label("Edit", systemImage: "pencil")
                                }
                                Button(role: .destructive) {
                                    Task { await deleteView(name: view.metadata?.name ?? "") }
                                } label: {
                                    Label("Delete", systemImage: "trash")
                                }
                            }
                        }
                    }
                }
                .navigationTitle("gitctl")
                .toolbar {
                    ToolbarItem(placement: .automatic) {
                        Button(action: { showCreateView = true }) {
                            Image(systemName: "plus")
                        }
                        .help("Add View")
                    }
                }
                .sheet(isPresented: $showCreateView) {
                    CreateViewSheet { newView in
                        Task {
                            do {
                                _ = try await client.createView(view: newView)
                                await loadViews()
                            } catch {
                                // TODO: show error
                            }
                        }
                    }
                }
                .sheet(item: $viewToEdit) { view in
                    EditViewSheet(view: view) { updatedView in
                        Task {
                            do {
                                _ = try await client.updateView(view: updatedView)
                                await loadViews()
                                // Update sidebar selection if the edited view is selected
                                if case .view(let v) = sidebarSelection, v.metadata?.name == updatedView.metadata?.name {
                                    sidebarSelection = .view(updatedView)
                                }
                            } catch {
                                // TODO: show error
                            }
                        }
                    }
                }
                .task { await loadViews() }
            } detail: {
                switch sidebarSelection {
                case .feed:
                    FeedListView(selectedItem: $selectedItem)
                case .assigned:
                    AssignedListView(selectedItem: $selectedItem)
                case .repos:
                    ReposView()
                case .view(let view):
                    ViewResultsListView(view: view, selectedItem: $selectedItem)
                case nil:
                    Text("Select a section")
                        .foregroundStyle(.secondary)
                }
            }
        }
    }

    func loadViews() async {
        do {
            views = try await client.listViews()
        } catch {
            // Silently handle — views section will be empty
        }
    }

    func deleteView(name: String) async {
        do {
            try await client.deleteView(name: name)
            await loadViews()
            // If the deleted view was selected, clear selection
            if case .view(let v) = sidebarSelection, v.metadata?.name == name {
                sidebarSelection = .feed
            }
        } catch {
            // TODO: show error
        }
    }
}

// MARK: - Create View Sheet

struct CreateViewSheet: SwiftUI.View {
    @Environment(\.dismiss) private var dismiss
    @State private var displayName = ""
    @State private var query = ""

    let onCreate: (View) -> Void

    var body: some SwiftUI.View {
        VStack(alignment: .leading, spacing: 16) {
            Text("New View")
                .font(.headline)

            TextField("Display Name", text: $displayName)
                .textFieldStyle(.roundedBorder)

            TextField("Query (e.g. is:pr is:open repo:org/repo author:@me)", text: $query)
                .textFieldStyle(.roundedBorder)

            HStack {
                Spacer()
                Button("Cancel") { dismiss() }
                    .keyboardShortcut(.cancelAction)
                Button("Create") {
                    let name = displayName.lowercased()
                        .replacingOccurrences(of: " ", with: "-")
                        .filter { $0.isLetter || $0.isNumber || $0 == "-" }
                    let view = View(
                        apiVersion: "gitctl.justinsb.com/v1alpha1",
                        kind: "View",
                        metadata: ObjectMeta(name: name),
                        spec: ViewSpec(query: query, displayName: displayName)
                    )
                    onCreate(view)
                    dismiss()
                }
                .keyboardShortcut(.defaultAction)
                .disabled(displayName.isEmpty || query.isEmpty)
            }
        }
        .padding()
        .frame(minWidth: 400)
    }
}

// MARK: - Edit View Sheet

struct EditViewSheet: SwiftUI.View {
    @Environment(\.dismiss) private var dismiss
    @State private var displayName: String
    @State private var query: String

    let view: View
    let onSave: (View) -> Void

    init(view: View, onSave: @escaping (View) -> Void) {
        self.view = view
        self.onSave = onSave
        _displayName = State(initialValue: view.spec?.displayName ?? "")
        _query = State(initialValue: view.spec?.query ?? "")
    }

    var body: some SwiftUI.View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Edit View")
                .font(.headline)

            TextField("Display Name", text: $displayName)
                .textFieldStyle(.roundedBorder)

            TextField("Query (e.g. is:pr is:open repo:org/repo author:@me)", text: $query)
                .textFieldStyle(.roundedBorder)

            HStack {
                Spacer()
                Button("Cancel") { dismiss() }
                    .keyboardShortcut(.cancelAction)
                Button("Save") {
                    var updated = view
                    updated.spec = ViewSpec(query: query, displayName: displayName)
                    onSave(updated)
                    dismiss()
                }
                .keyboardShortcut(.defaultAction)
                .disabled(displayName.isEmpty || query.isEmpty)
            }
        }
        .padding()
        .frame(minWidth: 400)
    }
}

// MARK: - View Results List View

struct ViewResultsListView: SwiftUI.View {
    let view: View
    @Binding var selectedItem: SelectableItem?
    @State private var prs: [PullRequest] = []
    @State private var issues: [Issue] = []
    @State private var isLoading = true
    @State private var errorMessage: String?
    @State private var searchText = ""

    private let client = GitCtlClient()

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

    var body: some SwiftUI.View {
        Group {
            if isLoading {
                ProgressView("Running query...")
            } else if let error = errorMessage {
                ErrorView(message: error) { Task { await load() } }
            } else {
                List(selection: $selectedItem) {
                    if !filteredPRs.isEmpty {
                        Section("Pull Requests (\(filteredPRs.count))") {
                            ForEach(filteredPRs) { pr in
                                PRRow(pr: pr)
                                    .tag(SelectableItem.pullRequest(pr))
                            }
                        }
                    }
                    if !filteredIssues.isEmpty {
                        Section("Issues (\(filteredIssues.count))") {
                            ForEach(filteredIssues) { issue in
                                IssueRow(issue: issue)
                                    .tag(SelectableItem.issue(issue))
                            }
                        }
                    }
                    if filteredPRs.isEmpty && filteredIssues.isEmpty {
                        Text("No results")
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
        .navigationTitle(view.spec?.displayName ?? "View")
        .searchable(text: $searchText, prompt: "Filter results")
        .toolbar {
            ToolbarItem(placement: .automatic) {
                Button(action: { Task { await load() } }) {
                    Image(systemName: "arrow.clockwise")
                }
                .help("Refresh")
            }
        }
        .task { await load() }
        .id(view.id) // force reload when switching views
    }

    func load() async {
        isLoading = true
        errorMessage = nil
        do {
            let results = try await client.executeView(name: view.metadata?.name ?? "")
            prs = results.pullRequests ?? []
            issues = results.issues ?? []
            isLoading = false
        } catch {
            errorMessage = "Failed to execute view: \(error.localizedDescription)\n\nMake sure the backend is running:\n  go run cmd/gitctl-backend/main.go"
            isLoading = false
        }
    }
}

// MARK: - Feed List View (Outbound PRs)

struct FeedListView: SwiftUI.View {
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

    var body: some SwiftUI.View {
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

struct AssignedListView: SwiftUI.View {
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

    var body: some SwiftUI.View {
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

struct ReposView: SwiftUI.View {
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

    var body: some SwiftUI.View {
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

struct PRDetailView: SwiftUI.View {
    let pr: PullRequest

    private var detailURL: URL {
        let repo = pr.status?.repo ?? ""
        let number = pr.status?.number ?? 0
        // repo is "owner/repo", split for URL path
        return URL(string: "http://localhost:8484/ui/repos/\(repo)/pulls/\(number)")!
    }

    var body: some SwiftUI.View {
        DetailWebView(url: detailURL)
            .navigationTitle("\(pr.status?.repo ?? "")#\(pr.status?.number ?? 0)")
    }
}

struct IssueDetailView: SwiftUI.View {
    let issue: Issue

    private var detailURL: URL {
        let repo = issue.status?.repo ?? ""
        let number = issue.status?.number ?? 0
        return URL(string: "http://localhost:8484/ui/repos/\(repo)/issues/\(number)")!
    }

    var body: some SwiftUI.View {
        DetailWebView(url: detailURL)
            .navigationTitle("\(issue.status?.repo ?? "")#\(issue.status?.number ?? 0)")
    }
}

// MARK: - Row Views

struct PRRow: SwiftUI.View {
    let pr: PullRequest

    var body: some SwiftUI.View {
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

struct IssueRow: SwiftUI.View {
    let issue: Issue

    var body: some SwiftUI.View {
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

struct RepoRow: SwiftUI.View {
    let repo: GitRepo

    var body: some SwiftUI.View {
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

struct ErrorView: SwiftUI.View {
    let message: String
    let retry: () -> Void

    var body: some SwiftUI.View {
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
