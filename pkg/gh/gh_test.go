package gh

import (
	"encoding/json"
	"testing"
)

func TestNew(t *testing.T) {
	cli := New()
	if cli == nil {
		t.Fatal("expected non-nil CLI")
	}
}

func TestParseIssue(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    Issue
		wantErr bool
	}{
		{
			name: "valid issue",
			data: `{"number": 42, "title": "Test Issue", "state": "open", "body": "Test body"}`,
			want: Issue{
				Number: 42,
				Title:  "Test Issue",
				State:  "open",
				Body:   "Test body",
			},
			wantErr: false,
		},
		{
			name: "minimal issue",
			data: `{"number": 1, "title": "Min"}`,
			want: Issue{
				Number: 1,
				Title:  "Min",
			},
			wantErr: false,
		},
		{
			name:    "invalid json",
			data:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue, err := ParseIssue([]byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if issue.Number != tt.want.Number {
				t.Errorf("Number: got %d, want %d", issue.Number, tt.want.Number)
			}
			if issue.Title != tt.want.Title {
				t.Errorf("Title: got %q, want %q", issue.Title, tt.want.Title)
			}
			if issue.State != tt.want.State {
				t.Errorf("State: got %q, want %q", issue.State, tt.want.State)
			}
			if issue.Body != tt.want.Body {
				t.Errorf("Body: got %q, want %q", issue.Body, tt.want.Body)
			}
		})
	}
}

func TestIssueJSON(t *testing.T) {
	issue := Issue{
		Number: 123,
		Title:  "Test",
		State:  "open",
		Body:   "Body text",
		Labels: []string{"bug", "help wanted"},
	}

	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed Issue
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Number != issue.Number {
		t.Errorf("Number mismatch")
	}
	if parsed.Title != issue.Title {
		t.Errorf("Title mismatch")
	}
	if len(parsed.Labels) != len(issue.Labels) {
		t.Errorf("Labels length mismatch")
	}
}

// MockCLI implements CLI interface for testing
type MockCLI struct {
	IssueCreateFn      func(repo, title, body string) (string, error)
	IssueViewFn        func(repo string, number int, fields []string) ([]byte, error)
	IssueViewByStrFn   func(repo, issue string, fields []string) ([]byte, error)
	IssueListFn        func(repo, state string, limit int) ([]byte, error)
	IssueDeleteFn      func(repo string, number int) error
	IssueDeleteByStrFn func(repo, issue string) error
	IssueCommentFn     func(repo, issue, body string) error
	RepoViewFn         func(repo string, fields []string) ([]byte, error)
	APIUserFn          func() (string, error)
	GraphQLFn          func(query string) ([]byte, error)
	GraphQLWithJQFn    func(query, jq string) ([]byte, error)
}

func (m *MockCLI) IssueCreate(repo, title, body string) (string, error) {
	if m.IssueCreateFn != nil {
		return m.IssueCreateFn(repo, title, body)
	}
	return "", nil
}

func (m *MockCLI) IssueView(repo string, number int, fields []string) ([]byte, error) {
	if m.IssueViewFn != nil {
		return m.IssueViewFn(repo, number, fields)
	}
	return nil, nil
}

func (m *MockCLI) IssueViewByStr(repo, issue string, fields []string) ([]byte, error) {
	if m.IssueViewByStrFn != nil {
		return m.IssueViewByStrFn(repo, issue, fields)
	}
	return nil, nil
}

func (m *MockCLI) IssueList(repo, state string, limit int) ([]byte, error) {
	if m.IssueListFn != nil {
		return m.IssueListFn(repo, state, limit)
	}
	return nil, nil
}

func (m *MockCLI) IssueDelete(repo string, number int) error {
	if m.IssueDeleteFn != nil {
		return m.IssueDeleteFn(repo, number)
	}
	return nil
}

func (m *MockCLI) IssueDeleteByStr(repo, issue string) error {
	if m.IssueDeleteByStrFn != nil {
		return m.IssueDeleteByStrFn(repo, issue)
	}
	return nil
}

func (m *MockCLI) IssueComment(repo, issue, body string) error {
	if m.IssueCommentFn != nil {
		return m.IssueCommentFn(repo, issue, body)
	}
	return nil
}

func (m *MockCLI) RepoView(repo string, fields []string) ([]byte, error) {
	if m.RepoViewFn != nil {
		return m.RepoViewFn(repo, fields)
	}
	return nil, nil
}

func (m *MockCLI) APIUser() (string, error) {
	if m.APIUserFn != nil {
		return m.APIUserFn()
	}
	return "", nil
}

func (m *MockCLI) GraphQL(query string) ([]byte, error) {
	if m.GraphQLFn != nil {
		return m.GraphQLFn(query)
	}
	return nil, nil
}

func (m *MockCLI) GraphQLWithJQ(query, jq string) ([]byte, error) {
	if m.GraphQLWithJQFn != nil {
		return m.GraphQLWithJQFn(query, jq)
	}
	return nil, nil
}

func TestMockCLIImplementsInterface(t *testing.T) {
	var _ CLI = (*MockCLI)(nil)
	var _ CLI = (*DefaultCLI)(nil)
}

func TestMockCLI(t *testing.T) {
	mock := &MockCLI{
		IssueCreateFn: func(repo, title, body string) (string, error) {
			return "https://github.com/owner/repo/issues/1", nil
		},
		IssueViewFn: func(repo string, number int, fields []string) ([]byte, error) {
			return []byte(`{"number": 1, "title": "Test"}`), nil
		},
	}

	url, err := mock.IssueCreate("owner/repo", "Title", "Body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://github.com/owner/repo/issues/1" {
		t.Errorf("unexpected URL: %s", url)
	}

	data, err := mock.IssueView("owner/repo", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"number": 1, "title": "Test"}` {
		t.Errorf("unexpected data: %s", data)
	}
}
