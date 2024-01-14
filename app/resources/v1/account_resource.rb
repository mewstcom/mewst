# typed: strict
# frozen_string_literal: true

class V1::AccountResource < V1::ApplicationResource
  sig { returns(User) }
  attr_reader :user

  sig { returns(OauthAccessToken) }
  attr_reader :oauth_access_token

  sig { params(user: User, profile: Profile, oauth_access_token: OauthAccessToken).void }
  def initialize(user:, profile:, oauth_access_token:)
    @user = user
    @profile = profile
    @oauth_access_token = oauth_access_token
  end

  sig { returns(V1::ProfileResource) }
  def profile
    V1::ProfileResource.new(viewer: nil, profile: @profile)
  end
end
