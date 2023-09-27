# typed: strict
# frozen_string_literal: true

class CreateSessionUseCase < ApplicationUseCase
  class Result < T::Struct
    const :oauth_access_token, OauthAccessToken
    const :profile, Profile
    const :user, User
    const :actor, Actor
  end

  sig { params(user: User).returns(Result) }
  def call(user:)
    actor = user.first_actor

    user.track_sign_in

    Result.new(
      oauth_access_token: user.first_actor.active_access_token.not_nil!,
      profile: actor.profile.not_nil!,
      user:,
      actor:
    )
  end
end
