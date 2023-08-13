# typed: strict
# frozen_string_literal: true

class Internal::AccountResource < Internal::ApplicationResource
  sig { params(oauth_access_token: OauthAccessToken, profile: Profile, user: User).void }
  def initialize(oauth_access_token:, profile:, user:)
    @oauth_access_token = oauth_access_token
    @profile = profile
    @user = user
  end

  sig { returns(OauthAccessToken) }
  attr_reader :oauth_access_token
  private :oauth_access_token

  sig { returns(Profile) }
  attr_reader :profile
  private :profile

  sig { returns(User) }
  attr_reader :user
  private :user
end
