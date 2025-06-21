package auth

import (
	"context"

	"strings"

	"github.com/tuongaz/go-saas/pkg/log"
)

// BootstrapAdmin checks for and creates a default admin user if one doesn't exist.

// This function is meant to be called during application initialization.

func (s *service) BootstrapAdmin(ctx context.Context) error {

	// Check if bootstrap admin is enabled

	adminEmail := s.cfg.BootstrapAdminEmail

	if adminEmail == "" {

		// Admin bootstrap is not configured, skip

		return nil

	}

	// Check if the admin user already exists

	exists, err := s.store.LoginCredentialsUserEmailExists(ctx, adminEmail)

	if err != nil {

		return err

	}

	if exists {

		// Admin user already exists, nothing to do

		log.Info("Admin user already exists, skipping bootstrap")

		return nil

	}

	// Get admin details from config

	adminPassword := s.cfg.BootstrapAdminPassword

	if adminPassword == "" {

		log.Default().Warn("Admin bootstrap password not set, skipping admin creation")

		return nil

	}

	adminName := s.cfg.BootstrapAdminName

	orgName := s.cfg.BootstrapAdminOrgName

	// Parse first and last name

	firstName, lastName := "Admin", "User"

	if adminName != "" {

		nameParts := strings.Fields(adminName)

		if len(nameParts) > 0 {

			firstName = nameParts[0]

			if len(nameParts) > 1 {

				lastName = strings.Join(nameParts[1:], " ")

			}

		}

	}

	// Create admin user and organization

	if orgName == "" {

		orgName = "Default Organization"

	}

	account, org, err := s.CreateAdminUserWithNewOrganisation(

		ctx, adminEmail, adminPassword, firstName, lastName, orgName)

	if err != nil {

		log.Error("Failed to create bootstrap admin user: %v", err)

		return err

	}

	log.Info("Created bootstrap admin user %s (%s) with organization %s (%s)",

		account.Name, account.ID, org.Name, org.ID)

	return nil

}
