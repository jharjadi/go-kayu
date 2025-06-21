package auth

import (
	"context"

	"fmt"

	"github.com/tuongaz/go-saas/core/auth/model"

	"github.com/tuongaz/go-saas/core/auth/store"
)

// CreateAdminUser creates a new user with admin role in a specified organization

func (s *service) CreateAdminUser(ctx context.Context, email, password, firstName, lastName string, organisationID string) (*model.Account, error) {

	// Check if user email already exists

	exists, err := s.store.LoginCredentialsUserEmailExists(ctx, email)

	if err != nil {

		return nil, fmt.Errorf("error checking if user exists: %w", err)

	}

	if exists {

		return nil, fmt.Errorf("user with email %s already exists", email)

	}

	// Create user credentials (for username/password auth)

	hashedPassword, err := s.hashPassword(password)

	if err != nil {

		return nil, fmt.Errorf("failed to hash password: %w", err)

	}

	// Get the organization to create the admin user for

	org, err := s.store.GetOrganisation(ctx, organisationID)

	if err != nil {

		return nil, fmt.Errorf("failed to get organisation: %w", err)

	}

	fullName := fmt.Sprintf("%s %s", firstName, lastName)

	// Use CreateOwnerAccount to create the account and associated records

	account, _, _, accountRole, err := s.store.CreateOwnerAccount(ctx, store.CreateOwnerAccountInput{

		Name: fullName,

		FirstName: firstName,

		LastName: lastName,

		Provider: model.AuthProviderUsernamePassword,

		Email: email,

		Password: hashedPassword,

		ProviderUserID: "", // This will be set within CreateOwnerAccount

	})

	if err != nil {

		return nil, fmt.Errorf("failed to create admin user: %w", err)

	}

	// If the account role is not for the requested organization, we need to update it

	if accountRole.OrganisationID != organisationID {

		// Create a new account role for the specified organization

		newAccountRole, err := s.store.AddOrganisationMember(ctx, store.AddOrganisationMemberInput{

			OrganisationID: organisationID,

			AccountID: account.ID,

			Role: string(model.RoleAdmin),
		})

		if err != nil {

			return nil, fmt.Errorf("failed to create admin role in organization: %w", err)

		}

		// If the organization doesn't have an owner, we need to create a new organization

		// since we can't directly update the owner_id field

		if org.OwnerID == "" {

			// Create a new organization with this user as owner

			// Handle organization properties safely (Description and Avatar are pointers)

			createOrgInput := store.CreateOrganisationInput{

				Name: org.Name,

				OwnerID: account.ID,
			}

			// Only set Description if it's not nil

			if org.Description != nil {

				description := *org.Description

				createOrgInput.Description = &description

			}

			// Only set Avatar if it's not nil

			if org.Avatar != nil {

				avatar := *org.Avatar

				createOrgInput.Avatar = &avatar

			}

			newOrg, err := s.store.CreateOrganisation(ctx, createOrgInput)

			if err != nil {

				return nil, fmt.Errorf("failed to create organization with new owner: %w", err)

			}

			// Now we have a new organization with the correct owner

			// We should transfer any existing members from the old organization

			members, err := s.store.ListOrganisationMembers(ctx, organisationID)

			if err != nil {

				return nil, fmt.Errorf("failed to list organization members: %w", err)

			}

			for _, member := range members {

				if member.AccountID != account.ID {

					_, err := s.store.AddOrganisationMember(ctx, store.AddOrganisationMemberInput{

						OrganisationID: newOrg.ID,

						AccountID: member.AccountID,

						Role: member.Role,
					})

					if err != nil {

						return nil, fmt.Errorf("failed to transfer member to new organization: %w", err)

					}

				}

			}

		}

		// For consistency, return the account with the role we just created

		accountRole = newAccountRole

	}

	return account, nil

}

// CreateAdminUserWithNewOrganisation creates both a new organization and a user with admin role in that organization

func (s *service) CreateAdminUserWithNewOrganisation(ctx context.Context, email, password, firstName, lastName, orgName string) (*model.Account, *model.Organisation, error) {

	// Check if user email already exists

	exists, err := s.store.LoginCredentialsUserEmailExists(ctx, email)

	if err != nil {

		return nil, nil, fmt.Errorf("error checking if user exists: %w", err)

	}

	if exists {

		return nil, nil, fmt.Errorf("user with email %s already exists", email)

	}

	// Create user credentials (for username/password auth)

	hashedPassword, err := s.hashPassword(password)

	if err != nil {

		return nil, nil, fmt.Errorf("failed to hash password: %w", err)

	}

	fullName := fmt.Sprintf("%s %s", firstName, lastName)

	// CreateOwnerAccount already creates an organization and sets the user as the owner

	account, org, _, _, err := s.store.CreateOwnerAccount(ctx, store.CreateOwnerAccountInput{

		Name: fullName,

		FirstName: firstName,

		LastName: lastName,

		Provider: model.AuthProviderUsernamePassword,

		Email: email,

		Password: hashedPassword,

		ProviderUserID: "", // This will be set within CreateOwnerAccount

	})

	if err != nil {

		return nil, nil, fmt.Errorf("failed to create admin user with organization: %w", err)

	}

	// Update organization name if it's different from the user's name

	if org.Name != orgName {

		updateInput := store.UpdateOrganisationInput{

			ID: org.ID,

			Name: &orgName,
		}

		_, err = s.store.UpdateOrganisation(ctx, updateInput)

		if err != nil {

			return nil, nil, fmt.Errorf("failed to update organization name: %w", err)

		}

		// Update the local reference

		org.Name = orgName

	}

	return account, org, nil

}
