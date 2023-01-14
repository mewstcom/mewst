# typed: strict
# frozen_string_literal: true

class TwitterAuthorization
  extend T::Sig

  include ActiveModel::Model
  include ActiveModel::Attributes
  include TwitterAuthorizable

  TWITTER_SCOPES = {
    # https://developer.twitter.com/en/docs/twitter-api/tweets/manage-tweets/api-reference/post-tweets
    cross_post: Mewst::TwitterOauth2::SCOPES.values_at(:tweet_read, :tweet_write, :users_read),
    # https://developer.twitter.com/en/docs/twitter-api/users/follows/api-reference/get-users-id-following
    find_friends: Mewst::TwitterOauth2::SCOPES.values_at(:tweet_read, :users_read, :follows_read)
  }.freeze

  attribute :cross_post_usage, :boolean
  attribute :find_friends_usage, :boolean

  def authorization_uri
    twitter_oauth2.client.authorization_uri(scope: scopes)
  end

  def code_verifier
    twitter_oauth2.client.code_verifier
  end

  def state
    twitter_oauth2.client.state
  end

  private

  def scopes
    cross_post_scopes = cross_post_usage ? TWITTER_SCOPES[:cross_post] : []
    find_friends_scopes = find_friends_usage ? TWITTER_SCOPES[:find_friends] : []

    (["offline.access"] + cross_post_scopes + find_friends_scopes).uniq
  end
end
