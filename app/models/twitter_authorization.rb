# typed: strict
# frozen_string_literal: true

class TwitterAuthorization
  extend T::Sig

  include ActiveModel::Model
  include ActiveModel::Attributes

  sig { returns(T.nilable(Profile)) }
  attr_accessor :profile

  attribute :cross_post_usage, :boolean
  attribute :find_friends_usage, :boolean

  def url
    twitter.client.authorization_uri(scope: scopes)
  end

  private

  sig { returns(Profile) }
  def profile!
    T.cast(profile, Profile)
  end

  def twitter
    @twitter ||= Twitter.new
  end

  def scopes
    cross_post_scopes = cross_post_usage ? Twitter::SCOPES.values_at(:tweet_write) : []
    find_friends_scopes = find_friends_usage ? Twitter::SCOPES.values_at(:follows_read) : []

    (cross_post_scopes + find_friends_scopes).uniq
  end
end
