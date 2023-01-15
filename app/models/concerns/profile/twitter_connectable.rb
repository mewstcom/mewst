# typed: true
# frozen_string_literal: true

module Profile::TwitterConnectable
  extend T::Sig
  extend ActiveSupport::Concern

  include TwitterAuthorizable

  sig { params(authorization_code: String, code_verifier: String).returns(T.self_type) }
  def upsert_twitter_account(authorization_code:, code_verifier:)
    access_token_responce = twitter_oauth2.create_access_token(authorization_code:, code_verifier:)

    ta = twitter_account.presence || build_twitter_account
    ta.reset_attributes(access_token_responce:)
    ta.save!

    ta
  end
end
