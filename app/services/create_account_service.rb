# typed: strict
# frozen_string_literal: true

class CreateAccountService < ApplicationService
  class Input < T::Struct
    extend T::Sig

    const :atname, String
    const :email, String
    const :locale, String
    const :password, String

    sig { params(form: Internal::AccountForm).returns(Input) }
    def self.from_internal_form(form:)
      new(
        atname: form.atname.not_nil!,
        email: form.email.not_nil!,
        locale: form.locale.not_nil!,
        password: form.password.not_nil!
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
    current_time = Time.current

    user = User.create!(
      email: input.email,
      locale: input.locale,
      password: input.password,
      signed_up_at: current_time
    )

    profile = user.create_profile!(
      atname: input.atname,
      joined_at: current_time
    )

    oauth_access_token = OauthAccessToken.find_or_create_for(
      application: OauthApplication.mewst_web,
      resource_owner: profile,
      scopes: "",
      user:
    )

    Result.new(oauth_access_token:, profile:, user:)
  end
end
