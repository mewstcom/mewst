# typed: strict
# frozen_string_literal: true

class CreateSessionService < ApplicationService
  class Input < T::Struct
    extend T::Sig

    const :user, User

    sig { params(form: Internal::SessionForm).returns(Input) }
    def self.from_internal_form(form:)
      new(
        user: form.user.not_nil!
      )
    end
  end

  class Result < T::Struct
    const :oauth_access_token, OauthAccessToken
    const :profile, Profile
    const :user, User
  end

  sig { params(input: Input).returns(Result) }
  def call(input:)
    user = input.user
    profile = user.profile.not_nil!

    user.track_sign_in

    Result.new(
      oauth_access_token: profile.active_access_token.not_nil!,
      profile:,
      user:
    )
  end
end
