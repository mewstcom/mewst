# typed: strict
# frozen_string_literal: true

class Services::CreateSession < Services::Base
  class Result < T::Struct
    const :oauth_access_token, OauthAccessToken
    const :profile, Profile
    const :user, User
  end

  sig { params(form: Forms::Session).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    user.track_sign_in

    Result.new(oauth_access_token:, profile:, user:)
  end

  private

  sig { returns(Forms::Session) }
  attr_reader :form

  sig { returns(User) }
  def user
    form.user!
  end

  sig { returns(Profile) }
  def profile
    user.first_profile
  end

  sig { returns(OauthAccessToken) }
  def oauth_access_token
    T.cast(OauthAccessToken.matching_token_for(OauthApplication.mewst_web, user, "", include_expired: false), OauthAccessToken)
  end
end
