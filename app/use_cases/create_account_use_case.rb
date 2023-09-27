# typed: strict
# frozen_string_literal: true

class CreateAccountUseCase < ApplicationUseCase
  class Result < T::Struct
    const :oauth_access_token, OauthAccessToken
    const :profile, Profile
    const :user, User
  end

  sig { params(atname: String, email: String, locale: String, password: String).returns(Result) }
  def call(atname:, email:, locale:, password:)
    account = Account.new(atname:, email:, locale:, password:)

    ActiveRecord::Base.transaction do
      account.save!

      Result.new(
        oauth_access_token: account.oauth_access_token.not_nil!,
        profile: account.profile.not_nil!,
        user: account.user.not_nil!
      )
    end
  end
end
