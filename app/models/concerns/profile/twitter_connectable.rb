# typed: true
# frozen_string_literal: true

module Profile::TwitterConnectable
  extend T::Sig
  extend ActiveSupport::Concern

  include TwitterAuthorizable

  sig { params(authorization_code: String, code_verifier: String).returns(T.self_type) }
  def upsert_twitter_account(authorization_code:, code_verifier:)
    access_token = twitter_oauth2.create_access_token(authorization_code:, code_verifier:)

    if twitter_account
      twitter_account.access_token = access_token.access_token
      return twitter_account.refresh
    end

    twitter_account = build_twitter_account(access_token: access_token.access_token)
    twitter_account.refresh
  end
end
