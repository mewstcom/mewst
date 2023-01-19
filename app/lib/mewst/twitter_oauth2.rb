# typed: strict
# frozen_string_literal: true

class Mewst::TwitterOauth2
  extend T::Sig

  SCOPES = T.let({
    follows_read: "follows.read",
    offline_access: "offline.access",
    tweet_read: "tweet.read",
    tweet_write: "tweet.write",
    users_read: "users.read"
  }.freeze, T::Hash[Symbol, String])

  class AccessTokenResponse < T::Struct
    const :access_token, String
    const :refresh_token, String
    const :scopes, T::Array[String]
    const :access_token_expired_at, Time
  end

  sig { params(client_id: String, client_secret: String, redirect_uri: String).void }
  def initialize(client_id:, client_secret:, redirect_uri:)
    @client_id = T.let(client_id, String)
    @client_secret = T.let(client_secret, String)
    @redirect_uri = T.let(redirect_uri, String)
  end

  sig { returns(TwitterOAuth2::Client) }
  def client
    @client ||= T.let(TwitterOAuth2::Client.new(
      identifier: @client_id,
      secret: @client_secret,
      redirect_uri: @redirect_uri
    ), T.nilable(TwitterOAuth2::Client))
  end

  sig { params(authorization_code: String, code_verifier: String).returns(AccessTokenResponse) }
  def create_access_token(authorization_code:, code_verifier:)
    client.authorization_code = authorization_code

    token_response = client.access_token!(code_verifier)
    access_token = token_response.access_token
    scopes = token_response.scope.split(" ")
    refresh_token = token_response.refresh_token
    access_token_expired_at = Time.current + token_response.expires_in

    AccessTokenResponse.new(access_token:, scopes:, refresh_token:, access_token_expired_at:)
  end
end
