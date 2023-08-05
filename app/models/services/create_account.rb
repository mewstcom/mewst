# typed: strict
# frozen_string_literal: true

class Services::CreateAccount < Services::Base
  class Result < T::Struct
    const :oauth_access_token, OauthAccessToken
    const :profile, Profile
    const :user, User
  end

  sig { params(form: Forms::Account).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    current_time = Time.current

    oauth_access_token, profile, user = ActiveRecord::Base.transaction do
      user = User.create!(
        email: form.email,
        locale: form.locale,
        password: form.password,
        signed_up_at: current_time
      )

      profile = user.create_profile!(
        atname: form.atname,
        joined_at: current_time
      )

      oauth_access_token = OauthAccessToken.find_or_create_for(
        application: OauthApplication.mewst_web,
        resource_owner: profile,
        scopes: "",
        user:
      )

      [oauth_access_token, profile, user]
    end

    Result.new(oauth_access_token:, profile:, user:)
  end

  sig { returns(Forms::Account) }
  attr_reader :form
  private :form
end
