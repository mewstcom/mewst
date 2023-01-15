# typed: strict
# frozen_string_literal: true

class Mewst::TwitterOauth2
  extend T::Sig

  SCOPES = {
    follows_read: "follows.read",
    tweet_read: "tweet.read",
    tweet_write: "tweet.write",
    users_read: "users.read"
  }.freeze

  class AccessTokenResponse < T::Struct
    const :access_token, String
    const :scopes, T::Array[String]
  end

  def initialize(client_id:, client_secret:, redirect_uri:)
    @client_id = client_id
    @client_secret = client_secret
    @redirect_uri = redirect_uri
  end

  def client
    @client ||= TwitterOAuth2::Client.new(
      identifier: @client_id,
      secret: @client_secret,
      redirect_uri: @redirect_uri
    )
  end

  def create_access_token(authorization_code:, code_verifier:)
    client.authorization_code = authorization_code

    token_response = client.access_token!(code_verifier)
    access_token = token_response.access_token
    scopes = token_response.scope.split(" ")

    AccessTokenResponse.new(access_token:, scopes:)
  end
end
