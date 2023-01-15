# typed: strict
# frozen_string_literal: true

class TwitterAccount < ApplicationRecord
  validate :permitted_scopes

  def can_cross_post?
    TwitterAuthorization::TWITTER_SCOPES[:cross_post].all? { |scope| scope.in?(scopes) }
  end

  def can_find_friends?
    TwitterAuthorization::TWITTER_SCOPES[:find_friends].all? { |scope| scope.in?(scopes) }
  end

  def reset_attributes(access_token_responce:)
    self.access_token = access_token_responce.access_token
    self.scopes = access_token_responce.scopes

    me = twitter_client.me

    self.uid = me.id
    self.username = me.username
  end

  private

  def permitted_scopes
    unpermitted_scopes = (scopes - Mewst::TwitterOauth2::SCOPES.values)

    return if unpermitted_scopes.empty?

    errors.add(:base, I18n.t(
      "activerecord.errors.models.twitter_account.attributes.base.permitted_scopes",
      unpermitted_scopes: unpermitted_scopes.join(", ")
    ))
  end

  def twitter_client
    @twitter_client = Mewst::TwitterClient.new(access_token:)
  end
end
