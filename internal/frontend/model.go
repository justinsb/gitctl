package frontend

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	pb "github.com/justinsb/gitctl/proto"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

type item struct {
	repo *pb.Repository
}

func (i item) FilterValue() string { return i.repo.FullName }

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
	str := fmt.Sprintf("%s", repo.FullName)
	if repo.Description != "" {
		str += fmt.Sprintf("\n    %s", repo.Description)
	}
	str += fmt.Sprintf("\n    ⭐ %d | 🍴 %d", repo.StargazersCount, repo.ForksCount)
	if repo.Language != "" {
		str += fmt.Sprintf(" | %s", repo.Language)
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type Model struct {
	client   pb.GitCtlClient
	username string
	list     list.Model
	repos    []*pb.Repository
	loading  bool
	err      error
}

type reposLoadedMsg struct {
	repos []*pb.Repository
}

type errMsg struct {
	err error
}

func NewModel(client pb.GitCtlClient, username string) Model {
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

func loadRepos(client pb.GitCtlClient, username string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := client.ListRepositories(ctx, &pb.ListRepositoriesRequest{
			Username: username,
		})
		if err != nil {
			return errMsg{err}
		}
		return reposLoadedMsg{repos: resp.Repositories}
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
