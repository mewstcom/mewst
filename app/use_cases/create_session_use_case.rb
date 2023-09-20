# typed: strict
# frozen_string_literal: true

class CreateSessionUseCase < ApplicationUseCase
  class Result < T::Struct
    const :oauth_access_token, OauthAccessToken
    const :profile, Profile
    const :user, User
  end

  sig { params(user, User).returns(Result) }
  def call(user:)
    profile = user.profile.not_nil!

    user.track_sign_in

    Result.new(
      oauth_access_token: profile.active_access_token.not_nil!,
      profile:,
      user:
    )
  end
end
