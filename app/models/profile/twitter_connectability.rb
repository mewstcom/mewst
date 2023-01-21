# typed: true
# frozen_string_literal: true

class Profile::TwitterConnectability
  extend T::Sig

  sig { params(profile: Profile).void }
  def initialize(profile:)
    @profile = profile
  end

  sig { params(authorization_code: String, code_verifier: String).returns(T.self_type) }
  def upsert_twitter_account(authorization_code:, code_verifier:)
    access_token_responce = profile.twitter_oauth2.create_access_token(authorization_code:, code_verifier:)

    ta = profile.twitter_account.presence || profile.build_twitter_account
    ta.reset_attributes(access_token_responce:)
    ta.save!

    ta
  end

  private

  sig { returns(Profile) }
  attr_reader :profile
end
