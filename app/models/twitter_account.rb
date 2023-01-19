# typed: strict
# frozen_string_literal: true

class TwitterAccount < ApplicationRecord
  belongs_to :profile

  validate :permitted_scopes

  sig { returns(Profile) }
  def profile!
    T.cast(profile, Profile)
  end

  sig { returns(String) }
  def profile_url
    "https://twitter.com/#{username}"
  end

  sig { returns(T::Boolean) }
  def can_cross_post?
    TwitterAuthorization::TWITTER_CROSS_POST_SCOPES.all? { |scope| scope.in?(scopes) }
  end

  sig { returns(T::Boolean) }
  def can_find_friends?
    TwitterAuthorization::TWITTER_FIND_FRIENDS_SCOPES.all? { |scope| scope.in?(scopes) }
  end

  sig { params(access_token_responce: Mewst::TwitterOauth2::AccessTokenResponse).returns(T.self_type) }
  def reset_attributes(access_token_responce:)
    self.access_token = access_token_responce.access_token
    self.scopes = access_token_responce.scopes
    self.refresh_token = access_token_responce.refresh_token
    self.access_token_expired_at = access_token_responce.access_token_expired_at

    me = twitter_client.me

    self.uid = me.id
    self.username = me.username

    self
  end

  sig { params(text: String).void }
  def tweet(text:)
    if access_token_expired_at.past?
      access_token_responce = profile!.twitter_oauth2.refresh_access_token(refresh_token:)
      reset_attributes(access_token_responce:)
      save!
    end

    twitter_client.tweet(text:)
  end

  private

  sig { void }
  def permitted_scopes
    unpermitted_scopes = (scopes - Mewst::TwitterOauth2::SCOPES.values)

    return if unpermitted_scopes.empty?

    errors.add(:base, I18n.t(
      "activerecord.errors.models.twitter_account.attributes.base.permitted_scopes",
      unpermitted_scopes: unpermitted_scopes.join(", ")
    ))
  end

  sig { returns(Mewst::TwitterClient) }
  def twitter_client
    Mewst::TwitterClient.new(access_token:)
  end
end
