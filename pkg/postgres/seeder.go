package postgres

import (
	"log"
	"time"

	"EquiliLearn/internal/entity"
	"EquiliLearn/pkg/bcrypt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SeedDefaultData populates default demo user accounts, sample groups, and messages.
func SeedDefaultData(db *gorm.DB) {
	bcryptService := bcrypt.NewBcrypt()
	hashedPassword, err := bcryptService.GenerateHash("Password123!")
	if err != nil {
		log.Printf("[Seeder] Error hashing default password: %v", err)
		return
	}

	// 1. Seed Demo Users
	demoUsers := []entity.User{
		{
			ID:        uuid.MustParse("a0000000-0000-0000-0000-000000000001"),
			Name:      "Alice Johnson",
			Email:     "alice@example.com",
			Password:  hashedPassword,
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.MustParse("a0000000-0000-0000-0000-000000000002"),
			Name:      "Bob Smith",
			Email:     "bob@example.com",
			Password:  hashedPassword,
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.MustParse("a0000000-0000-0000-0000-000000000003"),
			Name:      "Charlie Davis",
			Email:     "charlie@example.com",
			Password:  hashedPassword,
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, user := range demoUsers {
		var existing entity.User
		if err := db.Where("email = ?", user.Email).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if createErr := db.Create(&user).Error; createErr != nil {
					log.Printf("[Seeder] Failed to create seed user %s: %v", user.Email, createErr)
				} else {
					log.Printf("[Seeder] Created seed user: %s (%s)", user.Name, user.Email)
				}
			}
		} else {
			// Update password in case it was modified or not hashed
			db.Model(&existing).Updates(map[string]interface{}{
				"password": hashedPassword,
				"name":     user.Name,
			})
		}
	}

	// 2. Seed Default Demo Group
	groupID := uuid.MustParse("b0000000-0000-0000-0000-000000000001")
	var existingGroup entity.Group
	if err := db.Where("id = ?", groupID).First(&existingGroup).Error; err == gorm.ErrRecordNotFound {
		demoGroup := entity.Group{
			ID:          groupID,
			Name:        "EquiliLearn Study Group",
			Description: "Collaborative study room for EquiliLearn members to discuss assignments and share notes.",
			CreatedByID: demoUsers[0].ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := db.Create(&demoGroup).Error; err == nil {
			log.Printf("[Seeder] Created demo group: %s", demoGroup.Name)

			// Add Alice (admin), Bob (member), Charlie (member)
			members := []entity.GroupMember{
				{
					ID:       uuid.New(),
					GroupID:  groupID,
					UserID:   demoUsers[0].ID,
					Role:     "admin",
					JoinedAt: time.Now(),
				},
				{
					ID:       uuid.New(),
					GroupID:  groupID,
					UserID:   demoUsers[1].ID,
					Role:     "member",
					JoinedAt: time.Now(),
				},
				{
					ID:       uuid.New(),
					GroupID:  groupID,
					UserID:   demoUsers[2].ID,
					Role:     "member",
					JoinedAt: time.Now(),
				},
			}
			for _, m := range members {
				db.Create(&m)
			}

			// Add initial welcome messages
			messages := []entity.GroupMessage{
				{
					ID:          uuid.New(),
					GroupID:     groupID,
					SenderID:    demoUsers[0].ID,
					Content:     "Hello everyone! Welcome to the EquiliLearn Study Group 🎓",
					MessageType: "text",
					CreatedAt:   time.Now().Add(-10 * time.Minute),
					UpdatedAt:   time.Now().Add(-10 * time.Minute),
				},
				{
					ID:          uuid.New(),
					GroupID:     groupID,
					SenderID:    demoUsers[1].ID,
					Content:     "Hey Alice! Glad to be here. Did anyone review the latest lecture slides?",
					MessageType: "text",
					CreatedAt:   time.Now().Add(-5 * time.Minute),
					UpdatedAt:   time.Now().Add(-5 * time.Minute),
				},
			}
			for _, msg := range messages {
				db.Create(&msg)
			}
		}
	}
}
