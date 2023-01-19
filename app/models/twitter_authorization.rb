# typed: strict
# frozen_string_literal: true

class TwitterAuthorization
  extend T::Sig

  include ActiveModel::Model
  include ActiveModel::Attributes
  include TwitterAuthorizable

  # https://developer.twitter.com/en/docs/twitter-api/tweets/manage-tweets/api-reference/post-tweets
  TWITTER_CROSS_POST_SCOPES = T.let(
    Mewst::TwitterOauth2::SCOPES.values_at(:tweet_read, :tweet_write, :users_read).freeze,
    T::Array[String]
  )
  # https://developer.twitter.com/en/docs/twitter-api/users/follows/api-reference/get-users-id-following
  TWITTER_FIND_FRIENDS_SCOPES = T.let(
    Mewst::TwitterOauth2::SCOPES.values_at(:tweet_read, :users_read, :follows_read).freeze,
    T::Array[String]
  )

  attribute :cross_post_usage, :boolean
  attribute :find_friends_usage, :boolean

  sig { returns(String) }
  def authorization_uri
    twitter_oauth2.client.authorization_uri(scope: scopes)
  end

  sig { returns(String) }
  def code_verifier
    twitter_oauth2.client.code_verifier
  end

  sig { returns(String) }
  def state
    twitter_oauth2.client.state
  end

  private

  sig { returns(T::Array[String]) }
  def scopes
    # Add `offline_access` scope to issue a refresh token.
    # https://developer.twitter.com/en/docs/authentication/oauth-2-0/authorization-code
    default_scopes = T.cast([Mewst::TwitterOauth2::SCOPES[:offline_access]], T::Array[String])

    cross_post_scopes = cross_post_usage ? TWITTER_CROSS_POST_SCOPES : []
    find_friends_scopes = find_friends_usage ? TWITTER_FIND_FRIENDS_SCOPES : []

    (default_scopes + cross_post_scopes + find_friends_scopes).uniq
  end
end
