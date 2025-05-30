// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package commands

import (
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost/server/v8/cmd/mmctl/client"
	"github.com/mattermost/mattermost/server/v8/cmd/mmctl/printer"
)

func (s *MmctlE2ETestSuite) TestTokenGenerateForUserCmd() {
	s.SetupTestHelper().InitBasic()

	tokenDescription := model.NewRandomString(10)

	previousVal := s.th.App.Config().ServiceSettings.EnableUserAccessTokens
	s.th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableUserAccessTokens = true })
	defer s.th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableUserAccessTokens = *previousVal })

	s.RunForSystemAdminAndLocal("Generate token for user", func(c client.Client) {
		printer.Clean()

		user, appErr := s.th.App.CreateUser(s.th.Context, &model.User{Email: s.th.GenerateTestEmail(), Username: model.NewUsername(), Password: model.NewId()})
		s.Require().Nil(appErr)

		err := generateTokenForAUserCmdF(c, &cobra.Command{}, []string{user.Email, tokenDescription})
		s.Require().Nil(err)
		s.Require().Len(printer.GetErrorLines(), 0)

		userTokens, appErr := s.th.App.GetUserAccessTokensForUser(user.Id, 0, 1)
		s.Require().Nil(appErr)
		s.Require().Equal(1, len(userTokens))

		userToken, appErr := s.th.App.GetUserAccessToken(userTokens[0].Id, false)
		s.Require().Nil(appErr)

		expectedUserToken := printer.GetLines()[0].(*model.UserAccessToken)

		s.Require().Equal(expectedUserToken, userToken)
	})

	s.RunForSystemAdminAndLocal("Generate token for nonexistent user", func(c client.Client) {
		printer.Clean()

		nonExistentUserEmail := s.th.GenerateTestEmail()

		err := generateTokenForAUserCmdF(c, &cobra.Command{}, []string{nonExistentUserEmail, tokenDescription})
		s.Require().NotNil(err)
		s.Require().Len(printer.GetLines(), 0)
		s.Require().Len(printer.GetErrorLines(), 0)
		s.Require().Equal(
			fmt.Sprintf(`could not retrieve user information of %q`, nonExistentUserEmail),
			err.Error())
	})

	s.Run("Generate token without permission", func() {
		printer.Clean()

		user, appErr := s.th.App.CreateUser(s.th.Context, &model.User{Email: s.th.GenerateTestEmail(), Username: model.NewUsername(), Password: model.NewId()})
		s.Require().Nil(appErr)

		err := generateTokenForAUserCmdF(s.th.Client, &cobra.Command{}, []string{user.Email, tokenDescription})
		s.Require().NotNil(err)
		s.Require().Len(printer.GetLines(), 0)
		s.Require().Len(printer.GetErrorLines(), 0)
		s.Require().ErrorContains(
			err,
			fmt.Sprintf(`could not create token for %q: You do not have the appropriate permissions.`, user.Email),
		)
		userTokens, appErr := s.th.App.GetUserAccessTokensForUser(user.Id, 0, 1)
		s.Require().Nil(appErr)
		s.Require().Equal(0, len(userTokens))
	})
}

func (s *MmctlE2ETestSuite) TestRevokeTokenForAUserCmdF() {
	s.SetupTestHelper().InitBasic()

	previousVal := s.th.App.Config().ServiceSettings.EnableUserAccessTokens
	s.th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableUserAccessTokens = true })
	defer s.th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableUserAccessTokens = *previousVal })

	s.RunForSystemAdminAndLocal("Revoke user access tokens", func(c client.Client) {
		printer.Clean()

		user, appErr := s.th.App.CreateUser(s.th.Context, &model.User{Email: s.th.GenerateTestEmail(), Username: model.NewUsername(), Password: model.NewId()})
		s.Require().Nil(appErr)

		token, appErr := s.th.App.CreateUserAccessToken(s.th.Context, &model.UserAccessToken{
			UserId:      user.Id,
			Description: "test token",
		})
		s.Require().Nil(appErr)

		err := revokeTokenForAUserCmdF(c, &cobra.Command{}, []string{token.Id})
		s.Require().NoError(err)
		s.Require().Len(printer.GetLines(), 0)
		s.Require().Len(printer.GetErrorLines(), 0)

		_, appErr = s.th.App.GetUserAccessToken(token.Id, false)
		s.Require().Error(appErr)
		s.Require().Contains(appErr.Error(), "not found")
	})

	s.RunForSystemAdminAndLocal("Fail to revoke non-existent token", func(c client.Client) {
		printer.Clean()

		err := revokeTokenForAUserCmdF(c, &cobra.Command{}, []string{"non-existent-token-id"})
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "could not revoke token")
		s.Require().Len(printer.GetLines(), 0)
		s.Require().Len(printer.GetErrorLines(), 0)
	})

	s.Run("Revoke token without permission", func() {
		printer.Clean()

		user, appErr := s.th.App.CreateUser(s.th.Context, &model.User{Email: s.th.GenerateTestEmail(), Username: model.NewUsername(), Password: model.NewId()})
		s.Require().Nil(appErr)

		token, appErr := s.th.App.CreateUserAccessToken(s.th.Context, &model.UserAccessToken{
			UserId:      user.Id,
			Description: "test token",
		})
		s.Require().Nil(appErr)

		err := revokeTokenForAUserCmdF(s.th.Client, &cobra.Command{}, []string{token.Id})
		s.Require().Error(err)
		s.Require().Len(printer.GetLines(), 0)
		s.Require().Len(printer.GetErrorLines(), 0)
		s.Require().ErrorContains(
			err,
			fmt.Sprintf(`could not revoke token %q: You do not have the appropriate permissions.`, token.Id),
		)
	})
}
