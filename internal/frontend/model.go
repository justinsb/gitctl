package frontend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/justinsb/gitctl/internal/api"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	tabStyle          = lipgloss.NewStyle().Padding(0, 2)
	activeTabStyle    = lipgloss.NewStyle().Padding(0, 2).Bold(true).Foreground(lipgloss.Color("170"))
	tabBarStyle       = lipgloss.NewStyle().MarginLeft(2).MarginBottom(1)
	detailBorderStyle = lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				PaddingLeft(1)
	detailTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	detailMetaStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// Tab represents a screen/tab in the TUI.
type Tab int

const (
	TabFeed     Tab = iota // Outbound PRs
	TabAssigned            // Assigned PRs + Issues
	TabRepos               // Repositories
)

var tabNames = []string{"Feed", "Assigned", "Repos"}

// Client is a minimal HTTP client that communicates with the gitctl backend
// over a Unix domain socket using the Kubernetes wire protocol.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a Client that dials the backend over the given Unix socket path.
func NewClient(socketPath string) *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &Client{
		httpClient: &http.Client{Transport: transport},
		baseURL:    "http://localhost",
	}
}

// ListGitRepos calls GET /apis/.../gitrepos.
func (c *Client) ListGitRepos(ctx context.Context, username string) ([]api.GitRepo, error) {
	url := fmt.Sprintf("%s/apis/%s/%s/gitrepos?username=%s",
		c.baseURL, api.Group, api.Version, username)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to contact backend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("backend returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var repoList api.GitRepoList
	if err := json.NewDecoder(resp.Body).Decode(&repoList); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return repoList.Items, nil
}

// ListPullRequests calls GET /apis/.../pullrequests.
func (c *Client) ListPullRequests(ctx context.Context, username, scope string) ([]api.PullRequest, error) {
	url := fmt.Sprintf("%s/apis/%s/%s/pullrequests?username=%s&scope=%s",
		c.baseURL, api.Group, api.Version, username, scope)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to contact backend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("backend returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var prList api.PullRequestList
	if err := json.NewDecoder(resp.Body).Decode(&prList); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return prList.Items, nil
}

// ListIssues calls GET /apis/.../issues.
func (c *Client) ListIssues(ctx context.Context, username, scope string) ([]api.Issue, error) {
	url := fmt.Sprintf("%s/apis/%s/%s/issues?username=%s&scope=%s",
		c.baseURL, api.Group, api.Version, username, scope)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to contact backend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("backend returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var issueList api.IssueList
	if err := json.NewDecoder(resp.Body).Decode(&issueList); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return issueList.Items, nil
}

// ListComments calls GET /apis/.../comments.
func (c *Client) ListComments(ctx context.Context, repo string, number int) ([]api.Comment, error) {
	params := url.Values{}
	params.Set("repo", repo)
	params.Set("number", fmt.Sprintf("%d", number))
	reqURL := fmt.Sprintf("%s/apis/%s/%s/comments?%s",
		c.baseURL, api.Group, api.Version, params.Encode())
	log.Printf("ListComments: repo=%q number=%d url=%s", repo, number, reqURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to contact backend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("backend returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var commentList api.CommentList
	if err := json.NewDecoder(resp.Body).Decode(&commentList); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return commentList.Items, nil
}

// --- Repo list item (existing) ---

type repoItem struct {
	repo api.GitRepo
}

func (i repoItem) FilterValue() string { return i.repo.Status.FullName }

type repoDelegate struct{}

func (d repoDelegate) Height() int                             { return 3 }
func (d repoDelegate) Spacing() int                            { return 1 }
func (d repoDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d repoDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(repoItem)
	if !ok {
		return
	}

	repo := i.repo
	displayText := repo.Status.FullName
	if repo.Spec.Description != "" {
		displayText += fmt.Sprintf("\n    %s", repo.Spec.Description)
	}
	displayText += fmt.Sprintf("\n    * %d | fork %d", repo.Status.StargazersCount, repo.Status.ForksCount)
	if repo.Status.Language != "" {
		displayText += fmt.Sprintf(" | %s", repo.Status.Language)
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(displayText))
}

// --- PR list item ---

type prItem struct {
	pr api.PullRequest
}

func (i prItem) FilterValue() string {
	return fmt.Sprintf("%s#%d %s", i.pr.Status.Repo, i.pr.Status.Number, i.pr.Spec.Title)
}

type prDelegate struct{}

func (d prDelegate) Height() int                             { return 3 }
func (d prDelegate) Spacing() int                            { return 1 }
func (d prDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d prDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(prItem)
	if !ok {
		return
	}

	pr := i.pr
	title := fmt.Sprintf("%s#%d %s", pr.Status.Repo, pr.Status.Number, pr.Spec.Title)

	details := fmt.Sprintf("    %s", pr.Status.Author)
	if pr.Status.Draft {
		details += " [draft]"
	}
	if pr.Status.Merged {
		details += " [merged]"
	}
	if len(pr.Status.Labels) > 0 {
		details += " | " + strings.Join(pr.Status.Labels, ", ")
	}

	updated := ""
	if pr.Status.UpdatedAt != "" {
		updated = fmt.Sprintf("\n    updated %s", formatTime(pr.Status.UpdatedAt))
	}

	displayText := title + "\n" + details + updated

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(displayText))
}

// --- Issue list item ---

type issueItem struct {
	issue api.Issue
}

func (i issueItem) FilterValue() string {
	return fmt.Sprintf("%s#%d %s", i.issue.Status.Repo, i.issue.Status.Number, i.issue.Spec.Title)
}

type issueDelegate struct{}

func (d issueDelegate) Height() int                             { return 3 }
func (d issueDelegate) Spacing() int                            { return 1 }
func (d issueDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d issueDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(issueItem)
	if !ok {
		return
	}

	issue := i.issue
	title := fmt.Sprintf("%s#%d %s", issue.Status.Repo, issue.Status.Number, issue.Spec.Title)

	details := fmt.Sprintf("    %s", issue.Status.Author)
	if len(issue.Status.Labels) > 0 {
		details += " | " + strings.Join(issue.Status.Labels, ", ")
	}

	updated := ""
	if issue.Status.UpdatedAt != "" {
		updated = fmt.Sprintf("\n    updated %s", formatTime(issue.Status.UpdatedAt))
	}

	displayText := title + "\n" + details + updated

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(displayText))
}

// formatTime truncates an RFC 3339 timestamp to just the date for display.
func formatTime(t string) string {
	if len(t) >= 10 {
		return t[:10]
	}
	return t
}

// --- Messages ---

type reposLoadedMsg struct {
	repos []api.GitRepo
}

type feedLoadedMsg struct {
	prs []api.PullRequest
}

type assignedPRsLoadedMsg struct {
	prs []api.PullRequest
}

type assignedIssuesLoadedMsg struct {
	issues []api.Issue
}

type commentsLoadedMsg struct {
	comments []api.Comment
}

type errMsg struct {
	err error
}

// --- Model ---

// Model is the bubbletea model for the TUI with tabbed navigation.
type Model struct {
	client   *Client
	username string

	activeTab Tab
	width     int
	height    int

	// Feed tab (outbound PRs)
	feedList    list.Model
	feedLoading bool

	// Assigned tab (PRs + Issues combined)
	assignedList    list.Model
	assignedLoading bool

	// Repos tab
	repoList    list.Model
	repoLoading bool

	// Detail pane
	showDetail     bool
	detailFocused  bool
	detailViewport viewport.Model
	detailTitle    string
	detailContent  string
	detailComments []api.Comment
	detailLoading  bool
	// Track what's selected for detail
	selectedRepo   string
	selectedNumber int

	err error
}

// NewModel creates a new TUI Model using the given backend client and GitHub username.
func NewModel(client *Client, username string) Model {
	feedList := list.New([]list.Item{}, prDelegate{}, 0, 0)
	feedList.Title = "Feed - My Pull Requests"
	feedList.SetShowStatusBar(true)
	feedList.SetFilteringEnabled(true)
	feedList.Styles.Title = titleStyle
	feedList.Styles.PaginationStyle = paginationStyle
	feedList.Styles.HelpStyle = helpStyle

	assignedList := list.New([]list.Item{}, prDelegate{}, 0, 0)
	assignedList.Title = "Assigned to Me"
	assignedList.SetShowStatusBar(true)
	assignedList.SetFilteringEnabled(true)
	assignedList.Styles.Title = titleStyle
	assignedList.Styles.PaginationStyle = paginationStyle
	assignedList.Styles.HelpStyle = helpStyle

	repoList := list.New([]list.Item{}, repoDelegate{}, 0, 0)
	repoList.Title = fmt.Sprintf("Repositories for %s", username)
	repoList.SetShowStatusBar(true)
	repoList.SetFilteringEnabled(true)
	repoList.Styles.Title = titleStyle
	repoList.Styles.PaginationStyle = paginationStyle
	repoList.Styles.HelpStyle = helpStyle

	vp := viewport.New(0, 0)

	return Model{
		client:          client,
		username:        username,
		activeTab:       TabFeed,
		feedList:        feedList,
		feedLoading:     true,
		assignedList:    assignedList,
		assignedLoading: true,
		repoList:        repoList,
		repoLoading:     true,
		detailViewport:  vp,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadFeed(m.client, m.username),
		loadAssignedPRs(m.client, m.username),
		loadAssignedIssues(m.client, m.username),
		loadRepos(m.client, m.username),
	)
}

func loadFeed(client *Client, username string) tea.Cmd {
	return func() tea.Msg {
		prs, err := client.ListPullRequests(context.Background(), username, "outbound")
		if err != nil {
			return errMsg{err}
		}
		return feedLoadedMsg{prs: prs}
	}
}

func loadAssignedPRs(client *Client, username string) tea.Cmd {
	return func() tea.Msg {
		prs, err := client.ListPullRequests(context.Background(), username, "assigned")
		if err != nil {
			return errMsg{err}
		}
		return assignedPRsLoadedMsg{prs: prs}
	}
}

func loadAssignedIssues(client *Client, username string) tea.Cmd {
	return func() tea.Msg {
		issues, err := client.ListIssues(context.Background(), username, "assigned")
		if err != nil {
			return errMsg{err}
		}
		return assignedIssuesLoadedMsg{issues: issues}
	}
}

func loadRepos(client *Client, username string) tea.Cmd {
	return func() tea.Msg {
		repos, err := client.ListGitRepos(context.Background(), username)
		if err != nil {
			return errMsg{err}
		}
		return reposLoadedMsg{repos: repos}
	}
}

func loadComments(client *Client, repo string, number int) tea.Cmd {
	return func() tea.Msg {
		comments, err := client.ListComments(context.Background(), repo, number)
		if err != nil {
			return errMsg{err}
		}
		return commentsLoadedMsg{comments: comments}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizePanes()
		return m, nil

	case feedLoadedMsg:
		m.feedLoading = false
		items := make([]list.Item, len(msg.prs))
		for i, pr := range msg.prs {
			items[i] = prItem{pr: pr}
		}
		m.feedList.SetItems(items)
		return m, nil

	case assignedPRsLoadedMsg:
		m.assignedLoading = false
		// Combine assigned PRs with any existing assigned issues
		existing := m.assignedList.Items()
		items := make([]list.Item, 0, len(msg.prs)+len(existing))
		for _, pr := range msg.prs {
			items = append(items, prItem{pr: pr})
		}
		// Keep existing issue items
		for _, item := range existing {
			if _, ok := item.(issueItem); ok {
				items = append(items, item)
			}
		}
		m.assignedList.SetItems(items)
		return m, nil

	case assignedIssuesLoadedMsg:
		// Combine assigned issues with any existing assigned PRs
		existing := m.assignedList.Items()
		items := make([]list.Item, 0, len(msg.issues)+len(existing))
		// Keep existing PR items
		for _, item := range existing {
			if _, ok := item.(prItem); ok {
				items = append(items, item)
			}
		}
		for _, issue := range msg.issues {
			items = append(items, issueItem{issue: issue})
		}
		m.assignedList.SetItems(items)
		return m, nil

	case reposLoadedMsg:
		m.repoLoading = false
		items := make([]list.Item, len(msg.repos))
		for i, repo := range msg.repos {
			items[i] = repoItem{repo: repo}
		}
		m.repoList.SetItems(items)
		return m, nil

	case commentsLoadedMsg:
		m.detailLoading = false
		m.detailComments = msg.comments
		m.updateDetailViewport()
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.detailFocused {
				m.detailFocused = false
				return m, nil
			}
			if m.showDetail {
				m.showDetail = false
				m.detailFocused = false
				m.resizePanes()
				return m, nil
			}
			return m, tea.Quit
		case "ctrl+]", "tab":
			if !m.showDetail {
				m.activeTab = (m.activeTab + 1) % Tab(len(tabNames))
			}
			return m, nil
		case "shift+tab":
			if !m.showDetail {
				m.activeTab = (m.activeTab - 1 + Tab(len(tabNames))) % Tab(len(tabNames))
			}
			return m, nil
		case "enter":
			if m.showDetail && !m.detailFocused {
				// Focus the detail pane
				m.detailFocused = true
				return m, nil
			}
			if !m.showDetail {
				return m.openDetail()
			}
			return m, nil
		case "esc":
			if m.detailFocused {
				m.detailFocused = false
				return m, nil
			}
			if m.showDetail {
				m.showDetail = false
				m.resizePanes()
				return m, nil
			}
		}
	}

	// Route messages to the focused pane
	var cmd tea.Cmd
	if m.showDetail && m.detailFocused {
		m.detailViewport, cmd = m.detailViewport.Update(msg)
	} else {
		switch m.activeTab {
		case TabFeed:
			m.feedList, cmd = m.feedList.Update(msg)
		case TabAssigned:
			m.assignedList, cmd = m.assignedList.Update(msg)
		case TabRepos:
			m.repoList, cmd = m.repoList.Update(msg)
		}
	}
	return m, cmd
}

func (m Model) openDetail() (tea.Model, tea.Cmd) {
	var title, repo, author, state, body string
	var number int
	var labels []string

	switch m.activeTab {
	case TabFeed, TabAssigned:
		var activeList list.Model
		if m.activeTab == TabFeed {
			activeList = m.feedList
		} else {
			activeList = m.assignedList
		}
		selected := activeList.SelectedItem()
		if selected == nil {
			return m, nil
		}
		switch item := selected.(type) {
		case prItem:
			title = item.pr.Spec.Title
			repo = item.pr.Status.Repo
			number = item.pr.Status.Number
			author = item.pr.Status.Author
			state = item.pr.Status.State
			body = item.pr.Spec.Body
			labels = item.pr.Status.Labels
			if item.pr.Status.Draft {
				state = "draft"
			}
			if item.pr.Status.Merged {
				state = "merged"
			}
		case issueItem:
			title = item.issue.Spec.Title
			repo = item.issue.Status.Repo
			number = item.issue.Status.Number
			author = item.issue.Status.Author
			state = item.issue.Status.State
			body = item.issue.Spec.Body
			labels = item.issue.Status.Labels
		default:
			return m, nil
		}
	case TabRepos:
		// No detail view for repos yet
		return m, nil
	}

	m.showDetail = true
	m.detailFocused = true
	m.detailLoading = true
	m.detailComments = nil
	m.detailTitle = title
	m.selectedRepo = repo
	m.selectedNumber = number

	m.resizePanes()

	detailWidth := m.width / 2
	m.detailContent = buildDetailContent(title, repo, number, author, state, body, labels, nil, detailWidth)
	m.detailViewport.SetContent(m.detailContent)
	m.detailViewport.GotoTop()

	return m, loadComments(m.client, repo, number)
}

func (m *Model) updateDetailViewport() {
	// Rebuild content with stored item info + comments
	var title, repo, author, state, body string
	var number int
	var labels []string

	switch m.activeTab {
	case TabFeed, TabAssigned:
		var activeList list.Model
		if m.activeTab == TabFeed {
			activeList = m.feedList
		} else {
			activeList = m.assignedList
		}
		selected := activeList.SelectedItem()
		if selected != nil {
			switch item := selected.(type) {
			case prItem:
				title = item.pr.Spec.Title
				repo = item.pr.Status.Repo
				number = item.pr.Status.Number
				author = item.pr.Status.Author
				state = item.pr.Status.State
				body = item.pr.Spec.Body
				labels = item.pr.Status.Labels
			case issueItem:
				title = item.issue.Spec.Title
				repo = item.issue.Status.Repo
				number = item.issue.Status.Number
				author = item.issue.Status.Author
				state = item.issue.Status.State
				body = item.issue.Spec.Body
				labels = item.issue.Status.Labels
			}
		}
	}

	detailWidth := m.width / 2
	content := buildDetailContent(title, repo, number, author, state, body, labels, m.detailComments, detailWidth)
	m.detailViewport.SetContent(content)
}

func buildDetailContent(title, repo string, number int, author, state, body string, labels []string, comments []api.Comment, width int) string {
	var b strings.Builder

	// Header
	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("%s#%d", repo, number)))
	b.WriteString("\n")
	b.WriteString(detailTitleStyle.Render(title))
	b.WriteString("\n\n")

	// Meta
	meta := fmt.Sprintf("Author: %s  State: %s", author, state)
	if len(labels) > 0 {
		meta += "  Labels: " + strings.Join(labels, ", ")
	}
	b.WriteString(detailMetaStyle.Render(meta))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", minInt(width-2, 60)))
	b.WriteString("\n\n")

	// Body
	if body != "" {
		b.WriteString(body)
	} else {
		b.WriteString(detailMetaStyle.Render("(no description)"))
	}
	b.WriteString("\n")

	// Comments
	if len(comments) > 0 {
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", minInt(width-2, 60)))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("\n%d comment(s)\n", len(comments)))
		for _, c := range comments {
			b.WriteString("\n")
			b.WriteString(detailMetaStyle.Render(fmt.Sprintf("── %s  %s ──", c.Status.Author, formatTime(c.Status.CreatedAt))))
			b.WriteString("\n")
			b.WriteString(c.Spec.Body)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// resizePanes updates the list and detail viewport dimensions based on current state.
func (m *Model) resizePanes() {
	listHeight := m.height - 4
	listWidth := m.width
	if m.showDetail {
		listWidth = m.width / 2
		m.detailViewport.Width = m.width - listWidth - 2
		m.detailViewport.Height = listHeight
	}
	m.feedList.SetWidth(listWidth)
	m.feedList.SetHeight(listHeight)
	m.assignedList.SetWidth(listWidth)
	m.assignedList.SetHeight(listHeight)
	m.repoList.SetWidth(listWidth)
	m.repoList.SetHeight(listHeight)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n", m.err)
	}

	// Render tab bar
	var tabs []string
	for i, name := range tabNames {
		if Tab(i) == m.activeTab {
			tabs = append(tabs, activeTabStyle.Render("["+name+"]"))
		} else {
			tabs = append(tabs, tabStyle.Render(" "+name+" "))
		}
	}
	tabBar := tabBarStyle.Render(strings.Join(tabs, " "))

	// Render active tab content
	var content string
	switch m.activeTab {
	case TabFeed:
		if m.feedLoading {
			content = "\n  Loading feed...\n"
		} else {
			content = m.feedList.View()
		}
	case TabAssigned:
		if m.assignedLoading {
			content = "\n  Loading assigned items...\n"
		} else {
			content = m.assignedList.View()
		}
	case TabRepos:
		if m.repoLoading {
			content = "\n  Loading repositories...\n"
		} else {
			content = m.repoList.View()
		}
	}

	if m.showDetail {
		// Split view: list on left, detail on right
		listWidth := m.width / 2
		detailWidth := m.width - listWidth

		// Truncate list lines to listWidth
		listLines := strings.Split(content, "\n")
		detailStr := m.detailViewport.View()
		if m.detailLoading {
			detailStr = "Loading comments..."
		}
		borderStyle := detailBorderStyle
		if m.detailFocused {
			borderStyle = borderStyle.BorderForeground(lipgloss.Color("170"))
		}
		detailRendered := borderStyle.Width(detailWidth - 2).Render(detailStr)
		detailLines := strings.Split(detailRendered, "\n")

		// Join side by side
		maxLines := len(listLines)
		if len(detailLines) > maxLines {
			maxLines = len(detailLines)
		}

		var combined strings.Builder
		for i := 0; i < maxLines; i++ {
			left := ""
			if i < len(listLines) {
				left = listLines[i]
			}
			// Pad left to listWidth
			leftRunes := []rune(left)
			if len(leftRunes) > listWidth {
				leftRunes = leftRunes[:listWidth]
			}
			padded := string(leftRunes) + strings.Repeat(" ", maxInt(0, listWidth-len(leftRunes)))

			right := ""
			if i < len(detailLines) {
				right = detailLines[i]
			}
			combined.WriteString(padded)
			combined.WriteString(right)
			combined.WriteString("\n")
		}

		return "\n" + tabBar + "\n" + combined.String()
	}

	return "\n" + tabBar + "\n" + content
}
