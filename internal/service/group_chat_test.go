package service

import (
	"context"
	"testing"
	"time"

	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"

	"github.com/google/uuid"
)

type mockUserRepoForChat struct {
	users []entity.User
}

func (m *mockUserRepoForChat) CreateUser(ctx context.Context, user entity.User) error {
	m.users = append(m.users, user)
	return nil
}

func (m *mockUserRepoForChat) GetUser(ctx context.Context, pagination model.Pagination) ([]entity.User, error) {
	return m.users, nil
}

func (m *mockUserRepoForChat) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepoForChat) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepoForChat) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockUserRepoForChat) SaveResetToken(ctx context.Context, id uuid.UUID, token string, expired time.Time) error {
	return nil
}

func (m *mockUserRepoForChat) GetUserByResetToken(ctx context.Context, token string) (*entity.User, error) {
	return nil, nil
}

func (m *mockUserRepoForChat) UpdatePassword(ctx context.Context, id uuid.UUID, password string) error {
	return nil
}

func (m *mockUserRepoForChat) ClearResetToken(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockUserRepoForChat) SearchUsers(ctx context.Context, query string, limit int) ([]entity.User, error) {
	var matched []entity.User
	for _, u := range m.users {
		if query == u.Email || query == u.Name || query == u.ID.String() {
			matched = append(matched, u)
		}
	}
	return matched, nil
}

type mockGroupChatRepo struct {
	groups   []entity.Group
	members  []entity.GroupMember
	messages []entity.GroupMessage
}

func (m *mockGroupChatRepo) CreateGroup(ctx context.Context, group *entity.Group, memberUserIDs []uuid.UUID) error {
	m.groups = append(m.groups, *group)
	// Add creator as admin
	m.members = append(m.members, entity.GroupMember{
		ID:       uuid.New(),
		GroupID:  group.ID,
		UserID:   group.CreatedByID,
		Role:     "admin",
		JoinedAt: time.Now(),
	})
	for _, mid := range memberUserIDs {
		m.members = append(m.members, entity.GroupMember{
			ID:       uuid.New(),
			GroupID:  group.ID,
			UserID:   mid,
			Role:     "member",
			JoinedAt: time.Now(),
		})
	}
	return nil
}

func (m *mockGroupChatRepo) GetGroupByID(ctx context.Context, groupID uuid.UUID) (*entity.Group, error) {
	for _, g := range m.groups {
		if g.ID == groupID {
			groupCopy := g
			var gMembers []entity.GroupMember
			for _, mem := range m.members {
				if mem.GroupID == groupID {
					gMembers = append(gMembers, mem)
				}
			}
			groupCopy.Members = gMembers
			return &groupCopy, nil
		}
	}
	return nil, nil
}

func (m *mockGroupChatRepo) GetUserGroups(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]entity.Group, int64, error) {
	var userGroups []entity.Group
	for _, mem := range m.members {
		if mem.UserID == userID {
			for _, g := range m.groups {
				if g.ID == mem.GroupID {
					userGroups = append(userGroups, g)
				}
			}
		}
	}
	return userGroups, int64(len(userGroups)), nil
}

func (m *mockGroupChatRepo) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]entity.GroupMember, error) {
	var result []entity.GroupMember
	for _, mem := range m.members {
		if mem.GroupID == groupID {
			result = append(result, mem)
		}
	}
	return result, nil
}

func (m *mockGroupChatRepo) AddMember(ctx context.Context, member *entity.GroupMember) error {
	m.members = append(m.members, *member)
	return nil
}

func (m *mockGroupChatRepo) RemoveMember(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) error {
	var filtered []entity.GroupMember
	for _, mem := range m.members {
		if !(mem.GroupID == groupID && mem.UserID == userID) {
			filtered = append(filtered, mem)
		}
	}
	m.members = filtered
	return nil
}

func (m *mockGroupChatRepo) IsMember(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) (bool, string, error) {
	for _, mem := range m.members {
		if mem.GroupID == groupID && mem.UserID == userID {
			return true, mem.Role, nil
		}
	}
	return false, "", nil
}

func (m *mockGroupChatRepo) SaveMessage(ctx context.Context, msg *entity.GroupMessage) error {
	m.messages = append(m.messages, *msg)
	return nil
}

func (m *mockGroupChatRepo) GetGroupMessages(ctx context.Context, groupID uuid.UUID, pagination model.Pagination) ([]entity.GroupMessage, int64, error) {
	var result []entity.GroupMessage
	for _, msg := range m.messages {
		if msg.GroupID == groupID {
			result = append(result, msg)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockGroupChatRepo) GetLastGroupMessage(ctx context.Context, groupID uuid.UUID) (*entity.GroupMessage, error) {
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].GroupID == groupID {
			return &m.messages[i], nil
		}
	}
	return nil, nil
}

func TestGroupChatService_CreateGroupAndMembers(t *testing.T) {
	user1 := entity.User{ID: uuid.New(), Name: "Alice", Email: "alice@example.com"}
	user2 := entity.User{ID: uuid.New(), Name: "Bob", Email: "bob@example.com"}
	user3 := entity.User{ID: uuid.New(), Name: "Charlie", Email: "charlie@example.com"}

	userRepo := &mockUserRepoForChat{users: []entity.User{user1, user2, user3}}
	groupRepo := &mockGroupChatRepo{}
	svc := NewGroupChatService(groupRepo, userRepo)

	ctx := context.Background()

	// 1. Test SearchUsers
	searched, err := svc.SearchUsers(ctx, "bob@example.com")
	if err != nil || len(searched) != 1 || searched[0].ID != user2.ID {
		t.Fatalf("expected to find bob, got %v (err: %v)", searched, err)
	}

	// 2. Test CreateGroup with member email
	createReq := model.CreateGroupRequest{
		Name:         "Study Group 101",
		Description:  "Weekly prep",
		MemberEmails: []string{"bob@example.com"},
	}

	group, err := svc.CreateGroup(ctx, user1.ID, createReq)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	if group.Name != "Study Group 101" {
		t.Errorf("expected group name 'Study Group 101', got %s", group.Name)
	}

	if len(group.Members) != 2 {
		t.Fatalf("expected 2 members (Alice and Bob), got %d", len(group.Members))
	}

	// 3. Test AddMember (Charlie)
	addReq := model.AddGroupMemberRequest{
		Email: "charlie@example.com",
	}
	addedMember, err := svc.AddMember(ctx, group.ID, user1.ID, addReq)
	if err != nil {
		t.Fatalf("failed to add member Charlie: %v", err)
	}
	if addedMember.UserID != user3.ID {
		t.Errorf("expected member Charlie user id, got %v", addedMember.UserID)
	}

	// 4. Test SaveMessage & GetGroupMessages
	savedMsg, err := svc.SaveMessage(ctx, group.ID, user1.ID, "Hello team!", "text")
	if err != nil {
		t.Fatalf("failed to save message: %v", err)
	}
	if savedMsg.Content != "Hello team!" {
		t.Errorf("expected message content 'Hello team!', got %s", savedMsg.Content)
	}

	messages, total, err := svc.GetGroupMessages(ctx, group.ID, user2.ID, model.Pagination{Page: 1, Limit: 10})
	if err != nil || total != 1 || len(messages) != 1 {
		t.Fatalf("expected 1 message for Bob, got %d (err: %v)", total, err)
	}

	// 5. Test RemoveMember
	err = svc.RemoveMember(ctx, group.ID, user1.ID, user3.ID)
	if err != nil {
		t.Fatalf("failed to remove Charlie: %v", err)
	}

	isCharlieMember, _, _ := groupRepo.IsMember(ctx, group.ID, user3.ID)
	if isCharlieMember {
		t.Fatal("expected Charlie to no longer be a member")
	}
}
