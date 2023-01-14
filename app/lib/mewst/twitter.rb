# typed: strict
# frozen_string_literal: true

class Mewst::Twitter
  extend T::Sig

  SCOPES = {
    follows_read: "follows.read",
    tweet_write: "tweet.write",
    users_read: "users.read"
  }.freeze

  def client
    @client ||= TwitterOAuth2::Client.new(
      identifier: ENV.fetch("MEWST_TWITTER_CLIENT_ID"),
      secret: ENV.fetch("MEWST_TWITTER_CLIENT_SECRET"),
      redirect_uri: ENV.fetch("MEWST_TWITTER_REDIRECT_URI")
    )
  end

  def create_access_token(code:)
    binding.irb
    client.authorization_code = code
    client.access_token!(client.code_verifier)
  end
end
