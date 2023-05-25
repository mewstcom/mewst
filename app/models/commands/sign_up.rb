# typed: strict
# frozen_string_literal: true

class Commands::SignUp < Commands::Base
  class Result < T::Struct
    const :oauth_access_token, OauthAccessToken
  end

  sig { params(form: Forms::SignUp).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    oauth_access_token = T.let(nil, T.nilable(OauthAccessToken))
    current_time = Time.current

    ActiveRecord::Base.transaction do
      account = Account.create!(
        email: form.email,
        locale: form.locale,
        password: form.password,
        signed_up_at: current_time
      )
      profile = account.profiles.new(
        atname: form.atname,
        joined_at: current_time
      )
      account.profile_members.create!(profile:, joined_at: current_time)

      oauth_access_token = OauthAccessToken.find_or_create_for(
        application: OauthApplication.mewst_web,
        resource_owner: profile,
        scopes: "",
        account:
      )
    end

    Result.new(oauth_access_token:)
  end

  private

  sig { returns(Forms::SignUp) }
  attr_reader :form
end
