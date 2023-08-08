# typed: strict
# frozen_string_literal: true

class CreateAccountService::Result < T::Struct
  const :oauth_access_token, OauthAccessToken
  const :profile, Profile
  const :user, User
end
