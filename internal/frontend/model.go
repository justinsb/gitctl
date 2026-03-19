package frontend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/charmbracelet/bubbles/list"
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
)

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
		// The hostname is a placeholder; the socket dialer ignores it.
		baseURL: "http://localhost",
	}
}

// ListGitRepos calls GET /apis/gitctl.justinsb.com/v1alpha1/gitrepos and returns
// the parsed GitRepoList.
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

type item struct {
	repo api.GitRepo
}

func (i item) FilterValue() string { return i.repo.Status.FullName }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 3 }
func (d itemDelegate) Spacing() int                            { return 1 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	repo := i.repo
	displayText := repo.Status.FullName
	if repo.Spec.Description != "" {
		displayText += fmt.Sprintf("\n    %s", repo.Spec.Description)
	}
	displayText += fmt.Sprintf("\n    ⭐ %d | 🍴 %d", repo.Status.StargazersCount, repo.Status.ForksCount)
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

// Model is the bubbletea model for the repository list TUI.
type Model struct {
	client   *Client
	username string
	list     list.Model
	repos    []api.GitRepo
	loading  bool
	err      error
}

type reposLoadedMsg struct {
	repos []api.GitRepo
}

type errMsg struct {
	err error
}

// NewModel creates a new TUI Model using the given backend client and GitHub username.
func NewModel(client *Client, username string) Model {
	l := list.New([]list.Item{}, itemDelegate{}, 0, 0)
	l.Title = fmt.Sprintf("Repositories for %s", username)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return Model{
		client:   client,
		username: username,
		list:     l,
		loading:  true,
	}
}

func (m Model) Init() tea.Cmd {
	return loadRepos(m.client, m.username)
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil

	case reposLoadedMsg:
		m.loading = false
		m.repos = msg.repos
		items := make([]list.Item, len(msg.repos))
		for i, repo := range msg.repos {
			items[i] = item{repo: repo}
		}
		m.list.SetItems(items)
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.loading {
		return "\n  Loading repositories...\n"
	}
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n", m.err)
	}
	return "\n" + m.list.View()
}
