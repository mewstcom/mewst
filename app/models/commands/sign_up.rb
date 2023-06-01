# typed: strict
# frozen_string_literal: true

class Commands::SignUp < Commands::Base
  class Result < T::Struct
    const :oauth_access_token, OauthAccessToken
    const :profile, Profile
    const :user, User
  end

  sig { params(form: Forms::SignUp).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    user = T.let(nil, T.nilable(User))
    profile = T.let(nil, T.nilable(Profile))
    oauth_access_token = T.let(nil, T.nilable(OauthAccessToken))
    current_time = Time.current

    ActiveRecord::Base.transaction do
      user = User.create!(
        email: form.email,
        locale: form.locale,
        password: form.password,
        signed_up_at: current_time
      )
      profile = user.profiles.new(
        atname: form.atname,
        joined_at: current_time
      )
      user.profile_members.create!(profile:, joined_at: current_time)

      oauth_access_token = OauthAccessToken.find_or_create_for(
        application: OauthApplication.mewst_web,
        resource_owner: user,
        scopes: ""
      )
    end

    Result.new(oauth_access_token:, profile:, user:)
  end

  private

  sig { returns(Forms::SignUp) }
  attr_reader :form
end
